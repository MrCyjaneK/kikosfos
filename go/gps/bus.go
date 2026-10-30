package gps

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// refHold is the GeoClue reference. Hybris counts references per bus
// connection and drops them when that connection disappears, so closing
// the socket releases the GPS chip even if this process is killed.
type refHold interface {
	Hold(dest, path string) error
	Release()
}

type nopHold struct{}

func (nopHold) Hold(string, string) error { return fmt.Errorf("no session bus") }
func (nopHold) Release()                  {}

type liveHold struct {
	conn *dbusConn
	refs [][2]string
}

func (h *liveHold) Hold(dest, path string) error {
	if h == nil || h.conn == nil {
		return fmt.Errorf("no session bus")
	}
	if err := h.conn.call(dest, path, "org.freedesktop.Geoclue", "AddReference"); err != nil {
		return err
	}
	h.refs = append(h.refs, [2]string{dest, path})
	return nil
}

func (h *liveHold) Release() {
	if h == nil || h.conn == nil {
		return
	}
	for _, r := range h.refs {
		if err := h.conn.call(r[0], r[1], "org.freedesktop.Geoclue", "RemoveReference"); err != nil {
			log.Printf("kiko: gps: release %s: %v", r[0], err)
		}
	}
	h.refs = nil
	h.conn.Close()
	h.conn = nil
}

var dialHold = func() refHold {
	c, err := dialSession()
	if err != nil {
		log.Printf("kiko: gps: bus: %v", err)
		return nopHold{}
	}
	return &liveHold{conn: c}
}

type dbusConn struct {
	c    net.Conn
	next uint32
}

func dialSession() (*dbusConn, error) {
	sock := sessionSocket()
	if sock == "" {
		return nil, fmt.Errorf("no session bus socket")
	}
	c, err := net.DialTimeout("unix", sock, 2*time.Second)
	if err != nil {
		return nil, err
	}
	dc := &dbusConn{c: c, next: 1}
	if err := dc.auth(); err != nil {
		c.Close()
		return nil, err
	}
	if err := dc.call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "Hello"); err != nil {
		c.Close()
		return nil, err
	}
	return dc, nil
}

func (d *dbusConn) Close() {
	if d != nil && d.c != nil {
		d.c.Close()
		d.c = nil
	}
}

func (d *dbusConn) auth() error {
	if _, err := d.c.Write([]byte{0}); err != nil {
		return err
	}
	uid := hex.EncodeToString([]byte(strconv.Itoa(os.Getuid())))
	if _, err := fmt.Fprintf(d.c, "AUTH EXTERNAL %s\r\n", uid); err != nil {
		return err
	}
	for {
		line, err := readLine(d.c)
		if err != nil {
			return err
		}
		switch {
		case strings.HasPrefix(line, "OK"):
			_, err = io.WriteString(d.c, "BEGIN\r\n")
			return err
		case strings.HasPrefix(line, "DATA"):
			if _, err = io.WriteString(d.c, "DATA\r\n"); err != nil {
				return err
			}
		case strings.HasPrefix(line, "REJECTED"):
			return fmt.Errorf("dbus auth rejected")
		default:
			return fmt.Errorf("dbus auth: %s", line)
		}
	}
}

func (d *dbusConn) call(dest, path, iface, member string) error {
	return d.callBody(dest, path, iface, member, "", nil)
}

// Call is one method call. sig is the body signature, empty when there is no body.
func (d *dbusConn) Call(dest, path, iface, member, sig string, body []byte) error {
	if d == nil || d.c == nil {
		return fmt.Errorf("no session bus")
	}
	return d.callBody(dest, path, iface, member, sig, body)
}

func (d *dbusConn) callBody(dest, path, iface, member, sig string, body []byte) error {
	serial := d.next
	d.next++
	msg := encodeCallBody(serial, dest, path, iface, member, sig, body)
	d.c.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := d.c.Write(msg); err != nil {
		return err
	}
	for {
		m, err := readMsg(d.c)
		if err != nil {
			return err
		}
		if m.replyTo != serial || (m.kind != 2 && m.kind != 3) {
			continue
		}
		if m.kind == 3 {
			if m.errName == "" {
				return fmt.Errorf("dbus error")
			}
			return fmt.Errorf("%s", m.errName)
		}
		return nil
	}
}

func sessionSocket() string {
	addr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if i := strings.Index(addr, "unix:path="); i >= 0 {
		rest := addr[i+len("unix:path="):]
		if c := strings.IndexByte(rest, ','); c >= 0 {
			rest = rest[:c]
		}
		if rest != "" {
			return rest
		}
	}
	candidates := []string{
		fmt.Sprintf("/run/user/%d/dbus/user_bus_socket", os.Getuid()),
		"/run/user/100000/dbus/user_bus_socket",
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		candidates = append([]string{xdg + "/dbus/user_bus_socket"}, candidates...)
	}
	for _, sock := range candidates {
		if _, err := os.Stat(sock); err == nil {
			return sock
		}
	}
	return ""
}

func readLine(r io.Reader) (string, error) {
	var b []byte
	one := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, one); err != nil {
			return "", err
		}
		if one[0] == '\n' {
			return strings.TrimRight(string(b), "\r"), nil
		}
		b = append(b, one[0])
		if len(b) > 512 {
			return "", fmt.Errorf("dbus line too long")
		}
	}
}

type dbusMsg struct {
	kind    byte
	replyTo uint32
	errName string
}

func encodeCall(serial uint32, dest, path, iface, member string) []byte {
	return encodeCallBody(serial, dest, path, iface, member, "", nil)
}

func encodeCallBody(serial uint32, dest, path, iface, member, sig string, body []byte) []byte {
	var fields []byte
	fields = appendField(fields, 6, "s", dest)
	fields = appendField(fields, 1, "o", path)
	fields = appendField(fields, 2, "s", iface)
	fields = appendField(fields, 3, "s", member)
	if sig != "" {
		fields = appendField(fields, 8, "g", sig)
	}
	header := []byte{'l', 1, 0, 1}
	header = appendU32(header, uint32(len(body)))
	header = appendU32(header, serial)
	header = appendU32(header, uint32(len(fields)))
	header = append(header, fields...)
	header = align(header, 8)
	return append(header, body...)
}

// bodySDD is a D-Bus body of one string and two doubles.
func bodySDD(s string, a, b float64) []byte {
	buf := appendStr(nil, s)
	buf = align(buf, 8)
	buf = appendF64(buf, a)
	buf = align(buf, 8)
	buf = appendF64(buf, b)
	return buf
}

func appendF64(buf []byte, v float64) []byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
	return append(buf, b[:]...)
}

func appendField(buf []byte, code byte, sig, val string) []byte {
	buf = align(buf, 8)
	buf = append(buf, code)
	buf = append(buf, byte(len(sig)))
	buf = append(buf, sig...)
	buf = append(buf, 0)
	switch sig {
	case "s", "o":
		buf = appendStr(buf, val)
	case "g":
		buf = append(buf, byte(len(val)))
		buf = append(buf, val...)
		buf = append(buf, 0)
	}
	return buf
}

func appendStr(buf []byte, s string) []byte {
	buf = align(buf, 4)
	buf = appendU32(buf, uint32(len(s)))
	buf = append(buf, s...)
	buf = append(buf, 0)
	return buf
}

func appendU32(buf []byte, v uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return append(buf, b[:]...)
}

func align(buf []byte, n int) []byte {
	for len(buf)%n != 0 {
		buf = append(buf, 0)
	}
	return buf
}

func readMsg(r io.Reader) (dbusMsg, error) {
	var fixed [16]byte
	if _, err := io.ReadFull(r, fixed[:]); err != nil {
		return dbusMsg{}, err
	}
	if fixed[0] != 'l' {
		return dbusMsg{}, fmt.Errorf("dbus endian %q", fixed[0])
	}
	bodyLen := binary.LittleEndian.Uint32(fixed[4:8])
	fieldsLen := binary.LittleEndian.Uint32(fixed[12:16])
	fields := make([]byte, fieldsLen)
	if _, err := io.ReadFull(r, fields); err != nil {
		return dbusMsg{}, err
	}
	pad := (8 - (16+int(fieldsLen))%8) % 8
	if pad > 0 {
		if _, err := io.ReadFull(r, make([]byte, pad)); err != nil {
			return dbusMsg{}, err
		}
	}
	if bodyLen > 0 {
		if _, err := io.ReadFull(r, make([]byte, bodyLen)); err != nil {
			return dbusMsg{}, err
		}
	}
	reply, errName := parseHeader(fields)
	return dbusMsg{kind: fixed[1], replyTo: reply, errName: errName}, nil
}

func parseHeader(b []byte) (reply uint32, errName string) {
	i := 0
	for i < len(b) {
		for i < len(b) && i%8 != 0 {
			i++
		}
		if i >= len(b) {
			break
		}
		code := b[i]
		i++
		if i >= len(b) {
			break
		}
		sigLen := int(b[i])
		i++
		if i+sigLen >= len(b) {
			break
		}
		sig := string(b[i : i+sigLen])
		i += sigLen + 1
		switch sig {
		case "u":
			for i%4 != 0 {
				i++
			}
			if i+4 > len(b) {
				return
			}
			if code == 5 {
				reply = binary.LittleEndian.Uint32(b[i : i+4])
			}
			i += 4
		case "s", "o":
			for i%4 != 0 {
				i++
			}
			if i+4 > len(b) {
				return
			}
			n := int(binary.LittleEndian.Uint32(b[i : i+4]))
			i += 4
			if i+n > len(b) {
				return
			}
			if code == 4 {
				errName = string(b[i : i+n])
			}
			i += n + 1
		case "g":
			if i >= len(b) {
				return
			}
			n := int(b[i])
			i += 1 + n + 1
		default:
			return
		}
	}
	return
}
