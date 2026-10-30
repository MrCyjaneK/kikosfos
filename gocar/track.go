package main

import "math"

const (
	// maxAccM drops a fix that is too vague to add to the trip.
	maxAccM = 80
	// maxStepM is a teleport between one-second samples. The anchor moves
	// so the next real fix is not measured from the glitch.
	maxStepM = 250
)

type track struct {
	km       float64
	has      bool
	lat, lon float64
}

func (t *track) add(lat, lon, acc float64) {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsNaN(acc) {
		return
	}
	if acc > maxAccM {
		return
	}
	if !t.has {
		t.lat, t.lon = lat, lon
		t.has = true
		return
	}
	d := haversine(t.lat, t.lon, lat, lon)
	t.lat, t.lon = lat, lon
	if d > maxStepM {
		return
	}
	t.km += d / 1000
}

func (t *track) reset() {
	*t = track{}
}

// odometer counts only while a claimed ride is on screen.
// leave zeros it. A sample after leave does not count.
type odometer struct {
	track
	live bool
}

func (o *odometer) enter(claimed bool) {
	o.track.reset()
	o.live = claimed
}

func (o *odometer) leave() {
	o.track.reset()
	o.live = false
}

func (o *odometer) sample(lat, lon, acc float64, ok bool) {
	if !o.live || !ok {
		return
	}
	o.track.add(lat, lon, acc)
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earth = 6371000.0
	p1 := lat1 * math.Pi / 180
	p2 := lat2 * math.Pi / 180
	dlat := (lat2 - lat1) * math.Pi / 180
	dlon := (lon2 - lon1) * math.Pi / 180
	h := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * earth * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
