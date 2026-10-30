package main

import "testing"

func TestTrackSumsShortStep(t *testing.T) {
	var tr track
	tr.add(52, 21, 5)
	if tr.km != 0 || !tr.has {
		t.Fatalf("anchor %+v", tr)
	}
	tr.add(52.001, 21, 5)
	if tr.km < 0.10 || tr.km > 0.13 {
		t.Fatalf("km %v", tr.km)
	}
}

func TestTrackSkipsJumpAndBadAccuracy(t *testing.T) {
	var tr track
	tr.add(52, 21, 5)
	tr.add(53, 21, 5)
	if tr.km != 0 {
		t.Fatalf("jump counted %v", tr.km)
	}
	tr.add(53.001, 21, 200)
	if tr.km != 0 {
		t.Fatalf("vague fix counted %v", tr.km)
	}
	tr.add(53.001, 21, 5)
	if tr.km < 0.10 || tr.km > 0.13 {
		t.Fatalf("after glitch %v", tr.km)
	}
}

func TestOdometerLeaveResets(t *testing.T) {
	var o odometer
	o.enter(true)
	o.sample(52, 21, 5, true)
	o.sample(52.001, 21, 5, true)
	if o.km <= 0 {
		t.Fatal(o.km)
	}
	o.leave()
	if o.km != 0 || o.live || o.has {
		t.Fatalf("after leave %+v", o)
	}
	o.sample(52.002, 21, 5, true)
	if o.km != 0 || o.live {
		t.Fatalf("counted after leave %+v", o)
	}
}

func TestOdometerSkipsUnclaimed(t *testing.T) {
	var o odometer
	o.enter(false)
	o.sample(52, 21, 5, true)
	o.sample(52.001, 21, 5, true)
	if o.km != 0 || o.live {
		t.Fatalf("%+v", o)
	}
}
