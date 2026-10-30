package gps

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

type fakeBus struct {
	calls []fakeCall
	errAt int
	err   error
}

type fakeCall struct {
	dest, path, iface, member, sig string
	body                           []byte
}

func (f *fakeBus) Call(dest, path, iface, member, sig string, body []byte) error {
	f.calls = append(f.calls, fakeCall{dest, path, iface, member, sig, append([]byte{}, body...)})
	if f.err != nil && len(f.calls) == f.errAt {
		return f.err
	}
	return nil
}

func (f *fakeBus) Close() {}

func TestNavigateShowsPickupAndStarts(t *testing.T) {
	bus := &fakeBus{}
	prev := dialPure
	dialPure = func() (pureBus, error) { return bus, nil }
	t.Cleanup(func() { dialPure = prev })

	if err := Navigate(48.140833, 11.555, "Prielmayerstraße"); err != nil {
		t.Fatal(err)
	}
	if len(bus.calls) != 2 {
		t.Fatalf("calls = %d", len(bus.calls))
	}
	show := bus.calls[0]
	if show.member != "ShowPoi" || show.sig != "sdd" || show.iface != pureIface {
		t.Fatalf("show = %+v", show)
	}
	title, lat, lon := decodeSDD(t, show.body)
	if title != "Prielmayerstraße" || math.Abs(lat-48.140833) > 1e-9 || math.Abs(lon-11.555) > 1e-9 {
		t.Fatalf("body = %s %v %v", title, lat, lon)
	}
	start := bus.calls[1]
	if start.member != "Start" || start.path != pureNavPath || start.sig != "" {
		t.Fatalf("start = %+v", start)
	}
}

func TestNavigateOpensWhenMissing(t *testing.T) {
	bus := &fakeBus{errAt: 1, err: errors.New("org.freedesktop.DBus.Error.ServiceUnknown")}
	prev := dialPure
	dialPure = func() (pureBus, error) { return bus, nil }
	t.Cleanup(func() { dialPure = prev })
	prevWait := pureWait
	pureWait = 0
	t.Cleanup(func() { pureWait = prevWait })

	if err := Navigate(48.14, 11.555, ""); err != nil {
		t.Fatal(err)
	}
	if len(bus.calls) != 4 {
		t.Fatalf("calls = %d", len(bus.calls))
	}
	if bus.calls[0].member != "ShowPoi" || bus.calls[2].member != "ShowPoi" || bus.calls[3].member != "Start" {
		t.Fatalf("calls = %+v", bus.calls)
	}
	open := bus.calls[1]
	if open.member != "openUrl" || open.dest != lipDest || open.iface != lipIface || open.sig != "s" {
		t.Fatalf("open = %+v", open)
	}
	if got := decodeS(t, open.body); got != "geo:48.140000,11.555000" {
		t.Fatalf("geo = %s", got)
	}
}

func decodeS(t *testing.T, body []byte) string {
	t.Helper()
	if len(body) < 4 {
		t.Fatalf("short body %d", len(body))
	}
	n := int(binary.LittleEndian.Uint32(body[:4]))
	if 4+n > len(body) {
		t.Fatalf("string %d in %d", n, len(body))
	}
	return string(body[4 : 4+n])
}

func TestShowPoiMessageCarriesBody(t *testing.T) {
	body := bodySDD("Pickup", 48.140833, 11.555)
	msg := encodeCallBody(3, pureDest, purePath, pureIface, "ShowPoi", "sdd", body)
	if len(msg)%8 != 0 {
		t.Fatalf("len %d", len(msg))
	}
	if string(msg[len(msg)-len(body):]) != string(body) {
		t.Fatal("body is not at the end of the message")
	}
	title, lat, lon := decodeSDD(t, body)
	if title != "Pickup" || math.Abs(lat-48.140833) > 1e-9 || math.Abs(lon-11.555) > 1e-9 {
		t.Fatalf("%s %v %v", title, lat, lon)
	}
}

func decodeSDD(t *testing.T, body []byte) (string, float64, float64) {
	t.Helper()
	if len(body) < 4 {
		t.Fatalf("short body %d", len(body))
	}
	n := int(binary.LittleEndian.Uint32(body[:4]))
	if 4+n >= len(body) {
		t.Fatalf("string %d in %d", n, len(body))
	}
	title := string(body[4 : 4+n])
	i := 4 + n + 1
	for i%8 != 0 {
		i++
	}
	if i+16 > len(body) {
		t.Fatalf("doubles at %d len %d", i, len(body))
	}
	lat := math.Float64frombits(binary.LittleEndian.Uint64(body[i : i+8]))
	lon := math.Float64frombits(binary.LittleEndian.Uint64(body[i+8 : i+16]))
	return title, lat, lon
}
