package gps

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeCallRoundTrip(t *testing.T) {
	msg := encodeCall(7, hybrisDest, hybrisPath, "org.freedesktop.Geoclue", "AddReference")
	if len(msg)%8 != 0 {
		t.Fatalf("len %d", len(msg))
	}
	if msg[0] != 'l' || msg[1] != 1 {
		t.Fatalf("header %v", msg[:4])
	}
	got := string(msg)
	for _, part := range []string{hybrisDest, hybrisPath, "AddReference", "org.freedesktop.Geoclue"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %s", part)
		}
	}
	reply := encodeReply(7, "org.freedesktop.DBus.Error.Failed")
	m, err := readMsg(bytes.NewReader(reply))
	if err != nil {
		t.Fatal(err)
	}
	if m.kind != 3 || m.replyTo != 7 || m.errName != "org.freedesktop.DBus.Error.Failed" {
		t.Fatalf("%+v", m)
	}
}

func encodeReply(to uint32, errName string) []byte {
	var fields []byte
	fields = append(fields, 5, 1, 'u', 0)
	fields = appendU32(fields, to)
	fields = appendField(fields, 4, "s", errName)
	header := []byte{'l', 3, 0, 1}
	header = appendU32(header, 0)
	header = appendU32(header, 1)
	header = appendU32(header, uint32(len(fields)))
	header = append(header, fields...)
	return align(header, 8)
}

func TestStartHoldsProvidersAndStopReleases(t *testing.T) {
	h := &fakeHold{}
	prev := dialHold
	dialHold = func() refHold { return h }
	t.Cleanup(func() { dialHold = prev })
	s := &geoclueSource{}
	s.start()
	if !s.held || !s.net {
		t.Fatalf("held %v net %v", s.held, s.net)
	}
	if len(h.dests) != 2 || h.dests[0] != hybrisDest || h.dests[1] != mlsDest {
		t.Fatalf("%v", h.dests)
	}
	s.Close()
	if !h.released {
		t.Fatal("hold still open")
	}
}

type fakeHold struct {
	dests    []string
	released bool
}

func (f *fakeHold) Hold(dest, path string) error {
	f.dests = append(f.dests, dest)
	if path == "" {
		return fmtError("empty path")
	}
	return nil
}

func (f *fakeHold) Release() { f.released = true }

type fmtError string

func (e fmtError) Error() string { return string(e) }
