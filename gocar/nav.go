package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	pmDest    = "io.github.rinigus.PureMaps"
	pmPath    = "/io/github/rinigus/PureMaps/navigator"
	pmIface   = "io.github.rinigus.PureMaps.navigator"
	pmTimeout = 1500 * time.Millisecond

	// ErrNotRunning is the banner line when Pure Maps is not on the bus.
	ErrNotRunning = "Pure Maps not running"
)

// Trip is one navigator snapshot. Strings are Pure Maps' own text.
type Trip struct {
	OK                          bool
	Err                         string
	HasRoute                    bool
	Icon, ManDist               string
	Street, Narrative           string
	DestDist, DestTime, DestEta string
	RoundaboutExit              int
}

var runGdbus = gdbus

func pollPure() Trip {
	out, err := runGdbus("call", "--session",
		"--dest", pmDest,
		"--object-path", pmPath,
		"--method", "org.freedesktop.DBus.Properties.GetAll",
		pmIface)
	if err != nil {
		return Trip{Err: ErrNotRunning}
	}
	trip := parseTrip(out)
	if !trip.OK {
		trip.Err = ErrNotRunning
	}
	return trip
}

func parseTrip(out string) Trip {
	d := parseGVariantDict(out)
	if len(d) == 0 {
		return Trip{}
	}
	return Trip{
		OK:             true,
		HasRoute:       gBool(d, "hasRoute"),
		Icon:           gStr(d, "icon"),
		ManDist:        gStr(d, "manDist"),
		Street:         gStr(d, "street"),
		Narrative:      gStr(d, "narrative"),
		DestDist:       gStr(d, "destDist"),
		DestTime:       gStr(d, "destTime"),
		DestEta:        gStr(d, "destEta"),
		RoundaboutExit: gInt(d, "roundaboutExit"),
	}
}

func parseGVariantDict(out string) map[string]string {
	m := map[string]string{}
	i := 0
	for i < len(out) {
		j := strings.Index(out[i:], "'")
		if j < 0 {
			break
		}
		i += j
		key, next, ok := readQuoted(out, i)
		if !ok {
			i++
			continue
		}
		i = next
		i = skipSpace(out, i)
		if i >= len(out) || out[i] != ':' {
			continue
		}
		i++
		i = skipSpace(out, i)
		if i >= len(out) || out[i] != '<' {
			continue
		}
		end := strings.IndexByte(out[i:], '>')
		if end < 0 {
			break
		}
		m[key] = strings.TrimSpace(out[i+1 : i+end])
		i += end + 1
	}
	return m
}

func readQuoted(s string, i int) (string, int, bool) {
	if i >= len(s) || s[i] != '\'' {
		return "", i, false
	}
	i++
	var b strings.Builder
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '\'' {
			return b.String(), i + 1, true
		}
		b.WriteByte(s[i])
		i++
	}
	return "", i, false
}

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
		i++
	}
	return i
}

func gStr(d map[string]string, k string) string {
	v := strings.TrimSpace(d[k])
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		inner := v[1 : len(v)-1]
		return strings.ReplaceAll(inner, `\'`, "'")
	}
	return ""
}

func gBool(d map[string]string, k string) bool {
	return strings.TrimSpace(d[k]) == "true"
}

func gInt(d map[string]string, k string) int {
	v := strings.TrimSpace(d[k])
	v = strings.TrimPrefix(v, "int32 ")
	v = strings.TrimPrefix(v, "int64 ")
	v = strings.TrimPrefix(v, "uint32 ")
	n, _ := strconv.Atoi(v)
	return n
}

// Plain drops a blank Pure Maps field. "-" is the app's empty marker.
func Plain(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return ""
	}
	return s
}

// banner is the trip header. ok is false when Pure Maps has no route.
// time and dist stay separate so the screen can lay them out as fields.
func banner(t Trip) (head, sub, time, dist string, ok bool) {
	if !t.OK || !t.HasRoute {
		return "", "", "", "", false
	}
	head = Plain(t.ManDist)
	if head == "" {
		head = Plain(t.Icon)
	}
	sub = Plain(t.Narrative)
	if sub == "" {
		sub = Plain(t.Street)
	}
	return head, sub, Plain(t.DestTime), Plain(t.DestDist), true
}

func gdbus(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pmTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gdbus", args...)
	cmd.Env = sessionBusEnv()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := buf.String()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("gdbus timed out")
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

func sessionBusEnv() []string {
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("DBUS_SESSION_BUS_ADDRESS")) != "" {
		return env
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
			return append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path="+sock)
		}
	}
	return env
}
