// Package gps reads a position from GeoClue or from the mock driver.
// A new backend implements Source and registers itself with Register.
package gps

import "strings"

// NoFix is the speed card copy until a backend has a position delta.
const NoFix = "Establishing GPS"

// SatNote is one satellite lock or drop. The shell draws it and does not
// keep it. The first sample from a backend is a baseline and produces none.
type SatNote struct {
	Gain                       bool
	Kicker, Title, Sub, Detail string
	// Flag is a country code the shell can draw: us, ru, eu, jp, cn.
	Flag string
}

// Flash is the short top-bar line for this lock or drop.
func (n SatNote) Flash() string {
	if n.Kicker == "SAT UPDATE" {
		return "UPDATE " + n.Title
	}
	net := n.Sub
	for _, sep := range []string{" · ", " / "} {
		if i := strings.Index(net, sep); i >= 0 {
			net = net[:i]
			break
		}
	}
	verb := "LOST"
	if n.Gain {
		verb = "LOCK"
	}
	if net == "" {
		return verb + " " + n.Title
	}
	return verb + " " + n.Title + " " + net
}

// Fix is one snapshot. HasSpeed is false until two positions can be differenced,
// and whenever the backend has no fix. Notes are satellite lock/drop events
// since the previous Snapshot. Position, satellites, and velocity are also
// written to the daemon log by the backend.
type Fix struct {
	Kmh      int
	HasSpeed bool
	Err      string
	Notes    []SatNote
	// HasPos is a latitude and longitude the map can center on. AccM is the
	// horizontal accuracy in meters. Course is degrees clockwise from north,
	// and only set from the same GNSS pair that produced the speed.
	HasPos    bool
	Lat, Lon  float64
	AccM      float64
	HasCourse bool
	Course    float64
}

// Source is one GPS backend. Snapshot must not block. Close stops any
// background poll and may be called more than once.
type Source interface {
	Snapshot() Fix
	Close()
}

// Backend is one GPS driver.
type Backend struct {
	ID, Label string
}

// Backends is the settings list, in display order.
func Backends() []Backend {
	return []Backend{
		{ID: "mock", Label: "Mock"},
		{ID: "geoclue", Label: "GeoClue"},
	}
}

// Normalize maps a config value onto a known backend id.
// Empty and unknown values are geoclue.
func Normalize(kind string) string {
	kind = strings.TrimSpace(kind)
	for _, b := range Backends() {
		if strings.EqualFold(kind, b.ID) || strings.EqualFold(kind, b.Label) {
			return b.ID
		}
	}
	return "geoclue"
}

// Label is the settings caption for a backend id.
func Label(kind string) string {
	kind = Normalize(kind)
	for _, b := range Backends() {
		if b.ID == kind {
			return b.Label
		}
	}
	return "GeoClue"
}

var backends = map[string]func() Source{}

// Register adds a backend. A later registration replaces the same id.
func Register(id string, open func() Source) {
	if id == "" || open == nil {
		return
	}
	backends[id] = open
}

func init() {
	Register("mock", func() Source { return mockSource{} })
	Register("geoclue", openGeoclue)
}

// Open returns the named backend. Empty and unknown names are geoclue.
func Open(kind string) Source {
	kind = Normalize(kind)
	if open, ok := backends[kind]; ok && open != nil {
		return open()
	}
	return openGeoclue()
}

// München Hauptbahnhof. The mock driver stays here.
const (
	MockLat  = 48.140833
	MockLon  = 11.555000
	MockAccM = 15.0
)

type mockSource struct{}

// Snapshot is the station. It does not move.
func (mockSource) Snapshot() Fix {
	return Fix{HasPos: true, Lat: MockLat, Lon: MockLon, AccM: MockAccM}
}

func (mockSource) Close() {}
