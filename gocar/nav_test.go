package main

import (
	"strings"
	"testing"
)

const sampleGetAll = `({'alongRoute': <false>, 'destDist': <'12 km'>, 'destEta': <'15:42'>, 'destTime': <'18 min'>, 'direction': <0.0>, 'directionValid': <true>, 'hasRoute': <true>, 'icon': <'right'>, 'language': <'en'>, 'manDist': <'200 m'>, 'manTime': <'30 s'>, 'mode': <'car'>, 'narrative': <'Turn right onto High Street'>, 'progress': <40>, 'roundaboutExit': <0>, 'running': <true>, 'street': <'High Street'>, 'totalDist': <'12 km'>, 'totalTime': <'18 min'>},)`

func TestParseTrip(t *testing.T) {
	n := parseTrip(sampleGetAll)
	if !n.OK || !n.HasRoute {
		t.Fatalf("%+v", n)
	}
	if n.Narrative != "Turn right onto High Street" || n.Street != "High Street" {
		t.Fatalf("text %+v", n)
	}
	if n.ManDist != "200 m" || n.DestDist != "12 km" || n.DestTime != "18 min" || n.DestEta != "15:42" {
		t.Fatalf("dist %+v", n)
	}
	head, sub, time, dist, ok := banner(n)
	if !ok || head != "200 m" || sub != "Turn right onto High Street" || time != "18 min" || dist != "12 km" {
		t.Fatalf("banner %q %q %q %q %v", head, sub, time, dist, ok)
	}
	if n.Icon != "right" || n.RoundaboutExit != 0 {
		t.Fatalf("icon %+v", n)
	}
	exit := parseTrip(strings.Replace(sampleGetAll, "roundaboutExit': <0>", "roundaboutExit': <2>", 1))
	if exit.RoundaboutExit != 2 {
		t.Fatalf("exit %d", exit.RoundaboutExit)
	}
}

func TestParseTripIdle(t *testing.T) {
	out := `({'alongRoute': <false>, 'destDist': <'-'>, 'destEta': <''>, 'destTime': <'-'>, 'hasRoute': <false>, 'icon': <''>, 'manDist': <'-'>, 'narrative': <''>, 'street': <''>},)`
	n := parseTrip(out)
	if !n.OK || n.HasRoute || Plain(n.ManDist) != "" || Plain(n.DestTime) != "" || Plain(n.DestDist) != "" {
		t.Fatalf("%+v", n)
	}
	if _, _, _, _, ok := banner(n); ok {
		t.Fatal("idle banner")
	}
}

func TestPollError(t *testing.T) {
	prev := runGdbus
	t.Cleanup(func() { runGdbus = prev })
	runGdbus = func(args ...string) (string, error) {
		return "", errStub
	}
	tr := pollPure()
	if tr.Err != ErrNotRunning || tr.HasRoute {
		t.Fatalf("%+v", tr)
	}
	runGdbus = func(args ...string) (string, error) {
		return sampleGetAll, nil
	}
	tr = pollPure()
	if tr.Err != "" || !tr.HasRoute || tr.Street != "High Street" {
		t.Fatalf("after poll %+v", tr)
	}
}

type errStubType struct{}

func (errStubType) Error() string { return "down" }

var errStub error = errStubType{}
