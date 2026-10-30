package gps

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestNormalizeGPS(t *testing.T) {
	if Normalize("") != "geoclue" || Normalize("nope") != "geoclue" || Normalize("GeoClue") != "geoclue" {
		t.Fatal(Normalize(""))
	}
	if Normalize("mock") != "mock" || Normalize("Mock") != "mock" {
		t.Fatal(Normalize("Mock"))
	}
	mock := Open("mock").Snapshot()
	if !mock.HasPos || mock.HasSpeed || mock.Lat != MockLat || mock.Lon != MockLon || mock.AccM != MockAccM {
		t.Fatalf("mock %+v", mock)
	}
}

func TestParseNetworkFixWithoutAltitude(t *testing.T) {
	pos := `(3, 1793373171, 52.520000000000003, 13.404999999999999, -nan, (4, 25000.0, -nan))`
	p, ok := parsePosition(pos)
	if !ok || math.Abs(p.lat-52.52) > 1e-6 || math.Abs(p.lon-13.405) > 1e-6 {
		t.Fatalf("%+v %v", p, ok)
	}
	if p.hasAlt || p.accH != 25000 || p.accLevel != 4 || gnssFix(p) {
		t.Fatalf("%+v", p)
	}
}

func TestParseHybrisSample(t *testing.T) {
	pos := `(7, 1793143466, 50.247704999999996, 19.040343333333333, 324.80000000000001, (6, 17.399999618530273, 24.299999237060547))`
	p, ok := parsePosition(pos)
	if !ok || math.Abs(p.lat-50.247705) > 1e-5 || math.Abs(p.lon-19.040343) > 1e-5 {
		t.Fatalf("%+v %v", p, ok)
	}
	if !p.hasAlt || math.Abs(p.alt-324.8) > 0.1 || math.Abs(p.accH-17.4) > 0.1 {
		t.Fatalf("%+v", p)
	}
	if p.at.Unix() != 1793143466 {
		t.Fatal(p.at)
	}

	sats := `(2142102767, 12, 46, [27, 10, 23, 32, 18, 15, 84, 326, 307, 234, 214, 206], [(27, 78, 189, 24), (10, 73, 116, 32), (84, 28, 83, 39), (326, 37, 67, 34), (307, 21, 70, 33), (234, 53, 104, 34), (214, 47, 125, 29), (206, 22, 38, 29), (8, 38, 90, 0)])`
	got := formatSats(parseSatellites(sats))
	for _, name := range []string{"G27(el78 az189 snr24)", "R20(el28 az83 snr39)", "E26(el37 az67 snr34)", "E7(el21 az70 snr33)", "C34(el53 az104 snr34)", "C6(el22 az38 snr29)"} {
		if !strings.Contains(got, name) {
			t.Fatalf("missing %s in %s", name, got)
		}
	}
	if strings.Contains(got, "G8") {
		t.Fatalf("unused bird logged: %s", got)
	}
	if !strings.HasPrefix(got, "sats 12/46 ") {
		t.Fatal(got)
	}

	fields, speed, dir, vok := parseVelocity(`(3, 1793143466, 0.0, 0.0, -nan)`)
	if !vok || fields != 3 || speed != 0 || dir != 0 {
		t.Fatalf("vel %v %v %v %v", fields, speed, dir, vok)
	}
	if formatVelocity(fields, speed, dir, vok) != "vel 0.0 m/s dir 0" {
		t.Fatal(formatVelocity(fields, speed, dir, vok))
	}
}

func TestSpeedFromDelta(t *testing.T) {
	prev := runGdbus
	t.Cleanup(func() { runGdbus = prev })
	n := 0
	runGdbus = func(args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "GetPosition"):
			n++
			if n == 1 {
				return `(7, 1000, 50.0, 19.0, 10.0, (6, 5.0, 8.0))`, nil
			}
			// 0.0001° north is about 11.1 m, so one second is ~40 km/h.
			return `(7, 1001, 50.0001, 19.0, 10.0, (6, 5.0, 8.0))`, nil
		case strings.Contains(joined, "GetLastSatellite"):
			return `(1000, 1, 1, [27], [(27, 40, 10, 30)])`, nil
		case strings.Contains(joined, "GetVelocity"):
			return `(1, 1000, 1.5, 0.0, -nan)`, nil
		default:
			t.Fatalf("unexpected %s", joined)
			return "", nil
		}
	}
	s := &geoclueSource{}
	s.poll()
	first := s.Snapshot()
	if first.HasSpeed || first.HasCourse || !first.HasPos {
		t.Fatalf("first %+v", first)
	}
	if math.Abs(first.Lat-50) > 1e-6 || math.Abs(first.Lon-19) > 1e-6 || math.Abs(first.AccM-5) > 0.1 {
		t.Fatalf("first pos %+v", first)
	}
	s.poll()
	f := s.Snapshot()
	course := f.Course
	if course > 180 {
		course -= 360
	}
	if course < 0 {
		course = -course
	}
	if !f.HasSpeed || f.Kmh < 38 || f.Kmh > 42 || !f.HasCourse || course > 2 {
		t.Fatalf("%+v", f)
	}
	s.poll()
	s.poll()
	// Same second fix repeated: the stub stays on the second point, so the
	// third call is n==3 which still returns the 1001 point. dt is 0.
	// Force a no-fix and check the card drops.
	runGdbus = func(args ...string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "GetPosition") {
			return `(0, 0, 0.0, 0.0, 0.0, (0, 0.0, 0.0))`, nil
		}
		return `()`, nil
	}
	s.poll()
	lost := s.Snapshot()
	if lost.HasSpeed || lost.HasPos || lost.HasCourse || lost.Err == "" {
		t.Fatalf("lost fix %+v", lost)
	}
}

func TestSatNotesBaselineThenLockAndLoss(t *testing.T) {
	s := &geoclueSource{}
	base := satFix{used: 1, visible: 2, birds: []bird{
		{prn: 27, el: 78, az: 189, snr: 24, used: true},
		{prn: 8, el: 10, az: 20, snr: 0},
	}}
	if notes := s.notesFor(base, time.Unix(1000, 0)); notes != nil {
		t.Fatalf("baseline %+v", notes)
	}
	next := satFix{used: 2, visible: 2, birds: []bird{
		{prn: 27, el: 78, az: 189, snr: 24, used: true},
		{prn: 326, el: 37, az: 67, snr: 34, used: true},
	}}
	notes := s.notesFor(next, time.Unix(1001, 0))
	if len(notes) != 1 || !notes[0].Gain || notes[0].Title != "E26" {
		t.Fatalf("%+v", notes)
	}
	if notes[0].Kicker != "SAT LOCK  2/2" || notes[0].Sub != "Galileo | European Union" || notes[0].Detail != "EL 37    AZ 67    SNR 34" || notes[0].Flag != "eu" {
		t.Fatalf("%+v", notes[0])
	}
	lost := satFix{used: 1, visible: 1, birds: []bird{
		{prn: 326, el: 37, az: 67, snr: 34, used: true},
	}}
	notes = s.notesFor(lost, time.Unix(1002, 0))
	if len(notes) != 1 || notes[0].Gain || notes[0].Title != "G27" || notes[0].Kicker != "SAT LOST  1/1" || notes[0].Sub != "GPS | United States" || notes[0].Flag != "us" {
		t.Fatalf("%+v", notes)
	}
	// Bring G27 back, then drop it again inside the loss quiet window.
	_ = s.notesFor(base, time.Unix(1003, 0))
	notes = s.notesFor(lost, time.Unix(1004, 0))
	if len(notes) != 0 {
		t.Fatalf("repeat %+v", notes)
	}
}

func TestSatBurstKeepsEachBird(t *testing.T) {
	s := &geoclueSource{}
	_ = s.notesFor(satFix{birds: []bird{{prn: 1, used: true}}}, time.Unix(1, 0))
	var birds []bird
	for _, prn := range []int{1, 10, 84, 234} {
		birds = append(birds, bird{prn: prn, el: 10, az: 20, snr: 15, used: true})
	}
	notes := s.notesFor(satFix{used: 4, birds: birds}, time.Unix(2, 0))
	if len(notes) != 3 || notes[0].Title != "G10" || notes[1].Flag != "ru" || notes[2].Flag != "cn" {
		t.Fatalf("%+v", notes)
	}
}

func TestSatQuietLastsFiveMinutes(t *testing.T) {
	s := &geoclueSource{}
	t0 := time.Unix(0, 0)
	only := satFix{birds: []bird{{prn: 27, used: true}}}
	both := satFix{birds: []bird{
		{prn: 27, used: true},
		{prn: 326, el: 37, az: 67, snr: 34, used: true},
	}}
	if s.notesFor(only, t0) != nil {
		t.Fatal("baseline")
	}
	if notes := s.notesFor(both, t0.Add(time.Second)); len(notes) != 1 || !notes[0].Gain {
		t.Fatalf("first %+v", notes)
	}
	_ = s.notesFor(only, t0.Add(2*time.Second))
	if notes := s.notesFor(both, t0.Add(3*time.Minute)); len(notes) != 0 {
		t.Fatalf("inside window %+v", notes)
	}
	_ = s.notesFor(only, t0.Add(3*time.Minute+time.Second))
	notes := s.notesFor(both, t0.Add(6*time.Minute))
	if len(notes) != 1 || !notes[0].Gain || notes[0].Title != "E26" {
		t.Fatalf("after window %+v", notes)
	}
}

func TestGNSSGate(t *testing.T) {
	p := point{ok: true, lat: 50.2, lon: 19.0, accH: 17, accLevel: 6}
	net := point{ok: true, accH: 240, accLevel: 3}
	if !gnssFix(p) || gnssFix(net) {
		t.Fatal("accuracy gate")
	}
	note := SatNote{Gain: true, Title: "E26", Sub: "Galileo · European Union"}
	if note.Flash() != "LOCK E26 Galileo" {
		t.Fatal(note.Flash())
	}
}

func TestStationaryJitterIsZero(t *testing.T) {
	t0 := time.Unix(1000, 0)
	parked := []point{
		{at: t0, lat: 50, lon: 19, accH: 17, ok: true},
		{at: t0.Add(time.Second), lat: 50.00012, lon: 19, accH: 17, ok: true},
	}
	kmh, _, ok := speedFromTrack(parked)
	if !ok || kmh != 0 {
		t.Fatalf("inside accuracy %d %v", kmh, ok)
	}
	// 14m out and back is a one-sample spike, not motion.
	out := []point{
		{at: t0, lat: 50, lon: 19, accH: 5, ok: true},
		{at: t0.Add(time.Second), lat: 50.00013, lon: 19, accH: 5, ok: true},
		{at: t0.Add(2 * time.Second), lat: 50, lon: 19, accH: 5, ok: true},
	}
	kmh, _, ok = speedFromTrack(out)
	if !ok || kmh != 0 {
		t.Fatalf("there and back %d %v", kmh, ok)
	}
}

func TestHaversineNorth(t *testing.T) {
	a := point{lat: 50, lon: 19, at: time.Unix(1000, 0)}
	b := point{lat: 50.0001, lon: 19, at: time.Unix(1001, 0)}
	kmh, ok := speedKmh(a, b, time.Second)
	if !ok || kmh < 38 || kmh > 42 {
		t.Fatalf("%d %v", kmh, ok)
	}
}
