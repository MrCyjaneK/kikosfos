package gps

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Hybris is the GPS chip (position, satellites, velocity). Mlsdb is the
// network fix. Both are referenced from one session-bus connection that
// closes when the shell stops, and when the process dies. The GeoClue
// master client is not used: PositionStart pins Hybris for the life of
// geoclue-master, and RemoveReference on that client does not unpin it.

const (
	hybrisDest = "org.freedesktop.Geoclue.Providers.Hybris"
	hybrisPath = "/org/freedesktop/Geoclue/Providers/Hybris"
	mlsDest    = "org.freedesktop.Geoclue.Providers.Mlsdb"
	mlsPath    = "/org/freedesktop/Geoclue/Providers/Mlsdb"
	pollEvery  = time.Second
	callMax    = 1500 * time.Millisecond
	// A pair closer than this is the same tick. A longer gap is not a
	// current speed, so the next pair starts over.
	minDT = 800 * time.Millisecond
	maxDT = 15 * time.Second
)

const (
	fieldLat = 1
	fieldLon = 2
	fieldAlt = 4
	velSpeed = 1
	velDir   = 2

	// A fix tighter than this is GNSS. Coarser fixes are network or cell.
	gnssAccM = 80.0
)

var runGdbus = gdbus

type point struct {
	at        time.Time
	lat, lon  float64
	alt, accH float64
	accLevel  int
	hasAlt    bool
	ok        bool
}

type geoclueSource struct {
	mu        sync.Mutex
	fix       Fix
	track     []point
	hasSpeed  bool
	kmh       int
	hasCourse bool
	course    float64
	logged    string
	satLog    string
	velLog    string
	seen      map[int]bird
	satReady  bool
	quiet     map[quietKey]time.Time
	hold      refHold
	net       bool

	held bool
	once sync.Once
	stop chan struct{}
	done chan struct{}
}

func openGeoclue() Source {
	s := &geoclueSource{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go s.loop()
	return s
}

func (s *geoclueSource) loop() {
	defer close(s.done)
	s.start()
	s.poll()
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-tick.C:
			s.poll()
		}
	}
}

func (s *geoclueSource) poll() {
	if s.stop != nil {
		select {
		case <-s.stop:
			return
		default:
		}
	}
	p, ok := s.readPosition()
	if !ok {
		s.lose()
		return
	}
	var notes []SatNote
	var sats string
	if got, ok := s.readSatFix(); ok {
		notes = s.notesFor(got, time.Now())
		sats = formatSats(got)
	}
	vel := s.readVel()
	s.mu.Lock()
	if gnssFix(p) {
		s.noteSpeed(p)
	}
	fix := Fix{
		Kmh: s.kmh, HasSpeed: s.hasSpeed, Notes: notes,
		HasPos:    true,
		Lat:       p.lat,
		Lon:       p.lon,
		AccM:      p.accH,
		HasCourse: s.hasCourse,
		Course:    s.course,
	}
	s.mu.Unlock()
	s.logExtra(p, fix, sats, vel)
	s.set(fix)
}

func (s *geoclueSource) lose() {
	s.mu.Lock()
	s.track = nil
	s.hasSpeed = false
	s.hasCourse = false
	s.mu.Unlock()
	s.set(Fix{Err: "no gps"})
}

func (s *geoclueSource) start() {
	h := dialHold()
	s.hold = h
	if err := h.Hold(hybrisDest, hybrisPath); err != nil {
		log.Printf("kiko: gps: hold hybris: %v", err)
	} else {
		s.held = true
		log.Printf("kiko: gps: holding hybris on this bus connection")
	}
	if err := h.Hold(mlsDest, mlsPath); err != nil {
		log.Printf("kiko: gps: hold mlsdb: %v", err)
	} else {
		s.net = true
	}
}

// readPosition prefers a GNSS fix over a coarser network fix.
func (s *geoclueSource) readPosition() (point, bool) {
	var net, chip point
	if s.net {
		out, err := dbusCall(mlsDest, mlsPath, "org.freedesktop.Geoclue.Position.GetPosition")
		if err == nil {
			net, _ = parsePosition(out)
		}
	}
	out, err := callHybris("org.freedesktop.Geoclue.Position.GetPosition")
	if err == nil {
		chip, _ = parsePosition(out)
	}
	switch {
	case chip.ok && gnssFix(chip) && (!net.ok || !gnssFix(net) || chip.accH <= net.accH):
		return chip, true
	case net.ok:
		return net, true
	case chip.ok:
		return chip, true
	default:
		return point{}, false
	}
}

func gnssFix(p point) bool {
	if !p.ok {
		return false
	}
	if p.accH > 0 && p.accH <= gnssAccM {
		return true
	}
	return p.accLevel >= 5 && p.accH == 0
}

func (s *geoclueSource) readSatFix() (satFix, bool) {
	out, err := callHybris("org.freedesktop.Geoclue.Satellite.GetLastSatellite")
	if err != nil || strings.TrimSpace(out) == "" {
		return satFix{}, false
	}
	return parseSatellites(out), true
}

type quietKey struct {
	prn  int
	gain bool
}

// notesFor emits a lock or drop after the first sample. The same bird and
// the same direction stay quiet for five minutes, so the first wave shows
// and a flicker does not repeat.
func (s *geoclueSource) notesFor(f satFix, now time.Time) []SatNote {
	cur := map[int]bird{}
	for _, b := range f.birds {
		if b.used {
			cur[b.prn] = b
		}
	}
	if !s.satReady {
		s.seen = cur
		s.satReady = true
		return nil
	}
	if s.quiet == nil {
		s.quiet = map[quietKey]time.Time{}
	}
	var gains, losses []bird
	for prn, b := range cur {
		if _, ok := s.seen[prn]; !ok {
			gains = append(gains, b)
		}
	}
	for prn, b := range s.seen {
		if _, ok := cur[prn]; !ok {
			losses = append(losses, b)
		}
	}
	s.seen = cur
	sort.Slice(gains, func(i, j int) bool { return gains[i].prn < gains[j].prn })
	sort.Slice(losses, func(i, j int) bool { return losses[i].prn < losses[j].prn })
	const cool = 5 * time.Minute
	var notes []SatNote
	for _, b := range gains {
		k := quietKey{b.prn, true}
		if now.Before(s.quiet[k]) {
			continue
		}
		s.quiet[k] = now.Add(cool)
		notes = append(notes, noteFor(b, true, f.used, f.visible))
	}
	for _, b := range losses {
		k := quietKey{b.prn, false}
		if now.Before(s.quiet[k]) {
			continue
		}
		s.quiet[k] = now.Add(cool)
		notes = append(notes, noteFor(b, false, f.used, f.visible))
	}
	return notes
}

func noteFor(b bird, gain bool, used, visible int) SatNote {
	name, network, country := satMeta(b.prn)
	kicker := "SAT LOST"
	if gain {
		kicker = "SAT LOCK"
	}
	if used > 0 || visible > 0 {
		kicker = fmt.Sprintf("%s  %d/%d", kicker, used, visible)
	}
	sub := network
	if country != "" && country != "Augmentation" && country != "Unknown" {
		sub = network + " | " + country
	}
	return SatNote{
		Gain:   gain,
		Kicker: kicker,
		Title:  name,
		Sub:    sub,
		Detail: fmt.Sprintf("EL %d    AZ %d    SNR %d", b.el, b.az, b.snr),
		Flag:   flagCode(country),
	}
}

func flagCode(country string) string {
	switch country {
	case "United States":
		return "us"
	case "Russia":
		return "ru"
	case "European Union":
		return "eu"
	case "Japan":
		return "jp"
	case "China":
		return "cn"
	default:
		return ""
	}
}

func (s *geoclueSource) readVel() string {
	out, err := callHybris("org.freedesktop.Geoclue.Velocity.GetVelocity")
	if err != nil {
		return ""
	}
	return formatVelocity(parseVelocity(out))
}

func (s *geoclueSource) logExtra(p point, fix Fix, sats, vel string) {
	pos := formatPos(p, fix)
	if pos != s.logged {
		s.logged = pos
		log.Printf("kiko: gps: %s", pos)
	}
	if sats != "" && sats != s.satLog {
		s.satLog = sats
		log.Printf("kiko: gps: %s", sats)
	}
	if vel != "" && vel != s.velLog {
		s.velLog = vel
		log.Printf("kiko: gps: %s", vel)
	}
}

func (s *geoclueSource) set(f Fix) {
	s.mu.Lock()
	f.Notes = append(s.fix.Notes, f.Notes...)
	s.fix = f
	s.mu.Unlock()
}

func (s *geoclueSource) Snapshot() Fix {
	s.mu.Lock()
	f := s.fix
	s.fix.Notes = nil
	s.mu.Unlock()
	return f
}

func (s *geoclueSource) Close() {
	s.once.Do(func() {
		if s.stop != nil {
			close(s.stop)
		}
	})
	if s.done != nil {
		<-s.done
	}
	if s.hold != nil {
		s.hold.Release()
		s.hold = nil
		s.held = false
		s.net = false
	}
}

func callHybris(method string) (string, error) {
	return dbusCall(hybrisDest, hybrisPath, method)
}

func dbusCall(dest, path, method string, extra ...string) (string, error) {
	args := []string{"call", "--session", "--dest", dest, "--object-path", path, "--method", method}
	args = append(args, extra...)
	return runGdbus(args...)
}

func parseObjectPath(s string) string {
	m := regexp.MustCompile(`'(/[^']+)'`).FindStringSubmatch(s)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func gdbus(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callMax)
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

func parsePosition(s string) (point, bool) {
	s = stripTypes(s)
	// Altitude and the accuracy radii are nan when that provider has no value.
	num := `-?(?:\d+(?:\.\d+)?|nan)`
	re := regexp.MustCompile(`\(\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(` + num + `)\s*,\s*(` + num + `)\s*,\s*(` + num + `)\s*,\s*\(\s*(-?\d+)\s*,\s*(` + num + `)\s*,\s*(` + num + `)`)
	m := re.FindStringSubmatch(s)
	if len(m) != 9 {
		return point{}, false
	}
	fields, _ := strconv.Atoi(m[1])
	ts, _ := strconv.ParseInt(m[2], 10, 64)
	lat, errLat := strconv.ParseFloat(m[3], 64)
	lon, errLon := strconv.ParseFloat(m[4], 64)
	alt, _ := strconv.ParseFloat(m[5], 64)
	level, _ := strconv.Atoi(m[6])
	accH, _ := strconv.ParseFloat(m[7], 64)
	if errLat != nil || errLon != nil || math.IsNaN(lat) || math.IsNaN(lon) {
		return point{}, false
	}
	if math.IsNaN(accH) {
		accH = 0
	}
	if fields&fieldLat == 0 || fields&fieldLon == 0 || ts <= 0 {
		return point{}, false
	}
	return point{
		at:       time.Unix(ts, 0),
		lat:      lat,
		lon:      lon,
		alt:      alt,
		accH:     accH,
		accLevel: level,
		hasAlt:   fields&fieldAlt != 0 && !math.IsNaN(alt),
		ok:       true,
	}, true
}

type satFix struct {
	used, visible int
	birds         []bird
}

type bird struct {
	prn, el, az, snr int
	used             bool
}

func parseSatellites(s string) satFix {
	s = stripTypes(s)
	var out satFix
	head := regexp.MustCompile(`\(\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)`).FindStringSubmatch(s)
	if len(head) == 4 {
		out.used, _ = strconv.Atoi(head[2])
		out.visible, _ = strconv.Atoi(head[3])
	}
	used := map[int]bool{}
	if i := strings.IndexByte(s, '['); i >= 0 {
		if j := strings.IndexByte(s[i:], ']'); j >= 0 {
			for _, part := range strings.Split(s[i+1:i+j], ",") {
				n, err := strconv.Atoi(strings.TrimSpace(part))
				if err == nil {
					used[n] = true
				}
			}
		}
	}
	for _, m := range regexp.MustCompile(`\(\s*(\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)\s*\)`).FindAllStringSubmatch(s, -1) {
		prn, _ := strconv.Atoi(m[1])
		el, _ := strconv.Atoi(m[2])
		az, _ := strconv.Atoi(m[3])
		snr, _ := strconv.Atoi(m[4])
		out.birds = append(out.birds, bird{prn: prn, el: el, az: az, snr: snr, used: used[prn]})
	}
	return out
}

func formatSats(f satFix) string {
	if f.visible == 0 && len(f.birds) == 0 {
		return ""
	}
	var names []string
	for _, b := range f.birds {
		if !b.used {
			continue
		}
		names = append(names, fmt.Sprintf("%s(el%d az%d snr%d)", satName(b.prn), b.el, b.az, b.snr))
	}
	return fmt.Sprintf("sats %d/%d %s", f.used, f.visible, strings.Join(names, " "))
}

// satMeta maps the Android PRN space Hybris reports:
// GPS 1–32, SBAS 33–64, GLONASS 65–96, QZSS 193–200, BeiDou 201–263, Galileo 301–336.
func satMeta(prn int) (name, network, country string) {
	switch {
	case prn >= 1 && prn <= 32:
		return fmt.Sprintf("G%d", prn), "GPS", "United States"
	case prn >= 33 && prn <= 64:
		return fmt.Sprintf("S%d", prn), "SBAS", "Augmentation"
	case prn >= 65 && prn <= 96:
		return fmt.Sprintf("R%d", prn-64), "GLONASS", "Russia"
	case prn >= 193 && prn <= 200:
		return fmt.Sprintf("Q%d", prn-192), "QZSS", "Japan"
	case prn >= 201 && prn <= 263:
		return fmt.Sprintf("C%d", prn-200), "BeiDou", "China"
	case prn >= 301 && prn <= 336:
		return fmt.Sprintf("E%d", prn-300), "Galileo", "European Union"
	default:
		return fmt.Sprintf("PRN%d", prn), "GNSS", "Unknown"
	}
}

func satName(prn int) string {
	name, _, _ := satMeta(prn)
	return name
}

func parseVelocity(s string) (fields int, speed, dir float64, ok bool) {
	s = stripTypes(s)
	m := regexp.MustCompile(`\(\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*([^\s,]+)\s*,\s*([^\s,]+)\s*,\s*([^\s,\)]+)`).FindStringSubmatch(s)
	if len(m) != 6 {
		return 0, 0, 0, false
	}
	fields, _ = strconv.Atoi(m[1])
	speed, err1 := strconv.ParseFloat(m[3], 64)
	dir, err2 := strconv.ParseFloat(m[4], 64)
	if err1 != nil || err2 != nil || fields&velSpeed == 0 {
		return fields, 0, 0, false
	}
	return fields, speed, dir, true
}

func formatVelocity(fields int, speed, dir float64, ok bool) string {
	if !ok {
		return ""
	}
	if fields&velDir == 0 {
		return fmt.Sprintf("vel %.1f m/s", speed)
	}
	return fmt.Sprintf("vel %.1f m/s dir %.0f", speed, dir)
}

func formatPos(p point, fix Fix) string {
	alt := ""
	if p.hasAlt {
		alt = fmt.Sprintf(" alt %.0fm", p.alt)
	}
	acc := ""
	if p.accH > 0 {
		acc = fmt.Sprintf(" ±%.0fm", p.accH)
	}
	spd := ""
	if fix.HasSpeed {
		spd = fmt.Sprintf(" %d km/h", fix.Kmh)
	}
	return fmt.Sprintf("pos %.6f,%.6f%s%s%s", p.lat, p.lon, alt, acc, spd)
}

// noteSpeed records one GNSS sample. The displayed speed is the net move
// across the recent window. A shift that stays inside the reported accuracy
// is the receiver wandering while parked.
func (s *geoclueSource) noteSpeed(p point) {
	if len(s.track) > 0 {
		last := s.track[len(s.track)-1]
		dt := p.at.Sub(last.at)
		if dt >= 0 && dt < minDT {
			return
		}
		if dt < 0 || dt > maxDT {
			s.track = nil
			s.hasSpeed = false
			s.hasCourse = false
		}
	}
	s.track = append(s.track, p)
	cut := p.at.Add(-maxDT)
	i := 0
	for i < len(s.track) && s.track[i].at.Before(cut) {
		i++
	}
	if i > 0 {
		s.track = append([]point(nil), s.track[i:]...)
	}
	if len(s.track) > 8 {
		s.track = append([]point(nil), s.track[len(s.track)-8:]...)
	}
	kmh, course, ok := speedFromTrack(s.track)
	if !ok {
		return
	}
	s.kmh = kmh
	s.hasSpeed = true
	if kmh > 0 {
		s.course = course
		s.hasCourse = true
	}
}

func speedKmh(a, b point, dt time.Duration) (int, bool) {
	if dt <= 0 {
		return 0, false
	}
	sec := dt.Seconds()
	meters := haversine(a.lat, a.lon, b.lat, b.lon)
	kmh := meters / sec * 3.6
	if math.IsNaN(kmh) || math.IsInf(kmh, 0) || kmh < 0 || kmh > 300 {
		return 0, false
	}
	return int(math.Round(kmh)), true
}

// speedFromTrack is zero when the window's net move never leaves the
// accuracy circle, or when the path wanders out and back.
func speedFromTrack(pts []point) (kmh int, course float64, ok bool) {
	if len(pts) < 2 {
		return 0, 0, false
	}
	a := pts[0]
	b := pts[len(pts)-1]
	dt := b.at.Sub(a.at)
	if dt < minDT {
		return 0, 0, false
	}
	net := haversine(a.lat, a.lon, b.lat, b.lon)
	noise := a.accH
	if b.accH > noise {
		noise = b.accH
	}
	if net <= noise {
		return 0, 0, true
	}
	var path float64
	for i := 1; i < len(pts); i++ {
		path += haversine(pts[i-1].lat, pts[i-1].lon, pts[i].lat, pts[i].lon)
	}
	if path > noise && net < path*0.45 {
		return 0, 0, true
	}
	kmh, ok = speedKmh(a, b, dt)
	if !ok {
		return 0, 0, false
	}
	return kmh, bearingDeg(a, b), true
}

// bearingDeg is degrees clockwise from north, from a to b.
func bearingDeg(a, b point) float64 {
	φ1 := a.lat * math.Pi / 180
	φ2 := b.lat * math.Pi / 180
	dλ := (b.lon - a.lon) * math.Pi / 180
	y := math.Sin(dλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(dλ)
	deg := math.Atan2(y, x) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earth = 6371000.0
	r := func(d float64) float64 { return d * math.Pi / 180 }
	φ1, φ2 := r(lat1), r(lat2)
	dφ := r(lat2 - lat1)
	dλ := r(lon2 - lon1)
	h := math.Sin(dφ/2)*math.Sin(dφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	return 2 * earth * math.Asin(math.Min(1, math.Sqrt(h)))
}

var typeWord = regexp.MustCompile(`\b(?:int32|int64|uint32|uint64|double|boolean)\s+`)

func stripTypes(s string) string {
	return typeWord.ReplaceAllString(s, "")
}
