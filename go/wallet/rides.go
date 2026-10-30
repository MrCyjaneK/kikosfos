package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"kiko-sfos/gps"
)

// A pickup inside this window is a ride for now. Later pickups are scheduled.
const nowWindow = 30 * time.Minute

// A ride offer the driver can see before accepting. Prices are this
// driver's own fees for the server the offer came from.
type driveOffer struct {
	ID           string
	Server       string
	Coin         int
	StartAddress string
	EndAddress   string
	PickupKm     float64
	TripM        uint64
	When         time.Time
	StartLat     float64
	StartLon     float64
	Immediate    bool
	Fare         uint64
	Base         uint64
	PerKm        uint64
}

type readyServer struct {
	Host  string
	Coin  int
	Base  uint64
	PerKm uint64
}

type catalogRide struct {
	id, startAddress, endAddress string
	after                        time.Duration
	tripM                        uint64
	startLat, startLon           float64
}

var (
	driveMu     sync.Mutex
	driveReady  []readyServer
	driveOffers []driveOffer
)

func resetDrives() {
	driveMu.Lock()
	driveReady = nil
	driveOffers = nil
	driveMu.Unlock()
}

// registrationFailure is the home-screen line for a server error object.
// An empty object is success. A missing car or unset fees never reach it.
func registrationFailure(host string, body []byte) (string, bool) {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Error == "" {
		return "", false
	}
	return "registration failed with " + host + ": " + payload.Error, true
}

func skipRegister(err error) bool {
	if errors.Is(err, errIncompatible) {
		return true
	}
	switch err.Error() {
	case "car is incomplete", "fees are not set":
		return true
	default:
		return false
	}
}

// RegisterDrivers posts the driver to every server.
// The returned text is one "registration failed with <server>: <reason>"
// line per server error, or empty when nothing was rejected.
// Servers that are not ready to register are skipped and produce no line.
func RegisterDrivers() (string, error) {
	doc, err := loadProfile()
	if err != nil {
		return "", err
	}
	var lines []string
	var ready []readyServer
	for i, server := range doc.Servers {
		resp, err := RegisterDriver(server.Host)
		if err != nil {
			if skipRegister(err) {
				continue
			}
			return "", err
		}
		if line, ok := registrationFailure(server.Host, resp); ok {
			lines = append(lines, line)
			continue
		}
		priced, err := rememberServer(i, server.Host)
		if err != nil {
			if skipRegister(err) {
				continue
			}
			return "", err
		}
		ready = append(ready, priced)
	}
	driveMu.Lock()
	driveReady = ready
	driveMu.Unlock()
	return strings.Join(lines, "\n"), nil
}

func rememberServer(index int, host string) (readyServer, error) {
	baseText, err := ServerFeeBase(index)
	if err != nil {
		return readyServer{}, err
	}
	perText, err := ServerFeePerKm(index)
	if err != nil {
		return readyServer{}, err
	}
	if baseText == "" || perText == "" {
		return readyServer{}, errors.New("fees are not set")
	}
	base, err := strconv.ParseUint(baseText, 10, 64)
	if err != nil {
		return readyServer{}, errors.New("fees are not set")
	}
	perKm, err := strconv.ParseUint(perText, 10, 64)
	if err != nil {
		return readyServer{}, errors.New("fees are not set")
	}
	coin, err := ServerCoin(index)
	if err != nil {
		return readyServer{}, err
	}
	if coin < 0 {
		return readyServer{}, errIncompatible
	}
	return readyServer{Host: host, Coin: coin, Base: base, PerKm: perKm}, nil
}

// fareLovelace is max(base + ceil(perKm * km), 1_000_000).
// meters is the trip length. ceil is exact for whole meters.
func fareLovelace(base, perKm, meters uint64) uint64 {
	extra := new(big.Int).Mul(new(big.Int).SetUint64(perKm), new(big.Int).SetUint64(meters))
	extra.Add(extra, big.NewInt(999))
	extra.Quo(extra, big.NewInt(1000))
	sum := extra.Add(extra, new(big.Int).SetUint64(base))
	if sum.Cmp(big.NewInt(1000000)) < 0 {
		return 1000000
	}
	if !sum.IsUint64() {
		return math.MaxUint64
	}
	return sum.Uint64()
}

// RefreshDrives keeps the requested rides whose pickup is within rangeKm
// of the current fix. rangeKm <= 0 means 5. The list is empty when no
// server registered. No fix clears the list and returns that error.
func RefreshDrives(rangeKm float64) error {
	if rangeKm <= 0 {
		rangeKm = 5
	}
	text, err := GPSPosition()
	if err != nil {
		setOffers(nil)
		return err
	}
	lat, lon, err := parseFix(text)
	if err != nil {
		setOffers(nil)
		return err
	}
	setOffers(offersAround(readyCopy(), lat, lon, rangeKm))
	return nil
}

func parseFix(text string) (float64, float64, error) {
	parts := strings.Split(text, "\n")
	if len(parts) < 2 {
		return 0, 0, errors.New(gps.NoFix)
	}
	lat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, errors.New(gps.NoFix)
	}
	lon, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, errors.New(gps.NoFix)
	}
	return lat, lon, nil
}

func readyCopy() []readyServer {
	driveMu.Lock()
	defer driveMu.Unlock()
	out := make([]readyServer, len(driveReady))
	copy(out, driveReady)
	return out
}

func setOffers(rows []driveOffer) {
	driveMu.Lock()
	driveOffers = rows
	driveMu.Unlock()
}

func offersAround(ready []readyServer, lat, lon, rangeKm float64) []driveOffer {
	if len(ready) == 0 {
		return nil
	}
	catalog := mockCatalog()
	now := time.Now()
	rows := make([]driveOffer, 0, len(catalog))
	for i, ride := range catalog {
		pickup := haversineKm(lat, lon, ride.startLat, ride.startLon)
		if pickup > rangeKm {
			continue
		}
		server := ready[i%len(ready)]
		when := now.Add(ride.after)
		rows = append(rows, driveOffer{
			ID:           ride.id,
			Server:       server.Host,
			Coin:         server.Coin,
			StartAddress: ride.startAddress,
			EndAddress:   ride.endAddress,
			PickupKm:     pickup,
			TripM:        ride.tripM,
			When:         when,
			StartLat:     ride.startLat,
			StartLon:     ride.startLon,
			Immediate:    !when.After(now.Add(nowWindow)),
			Fare:         fareLovelace(server.Base, server.PerKm, ride.tripM),
			Base:         server.Base,
			PerKm:        server.PerKm,
		})
	}
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		j := i
		for j > 0 && driveBefore(row, rows[j-1]) {
			rows[j] = rows[j-1]
			j--
		}
		rows[j] = row
	}
	return rows
}

// driveBefore orders rides for now ahead of later ones, nearest pickup first.
func driveBefore(a, b driveOffer) bool {
	if a.Immediate != b.Immediate {
		return a.Immediate
	}
	if a.PickupKm != b.PickupKm {
		return a.PickupKm < b.PickupKm
	}
	return a.When.Before(b.When)
}

func mockCatalog() []catalogRide {
	specs := []struct {
		id, start, end string
		after          time.Duration
		pickupM, tripM uint64
		bearing        float64
	}{
		{"9f2c01", "Prielmayerstraße", "Karlsplatz", 5 * time.Minute, 400, 1600, 0},
		{"9f2c02", "Stachus", "Odeonsplatz", 12 * time.Minute, 1200, 3000, 90},
		{"9f2c03", "Sendlinger Tor", "Münchner Freiheit", 3 * time.Hour, 2500, 4200, 45},
		{"9f2c04", "Theresienwiese", "Hauptbahnhof", 26 * time.Hour, 4800, 2100, 180},
		{"9f2c05", "Nymphenburg", "Marienplatz", 8 * time.Minute, 7000, 8000, 270},
		{"9f2c06", "Garching", "Hauptbahnhof", 50 * time.Hour, 12000, 15000, 10},
	}
	out := make([]catalogRide, len(specs))
	for i, spec := range specs {
		lat, lon := destination(gps.MockLat, gps.MockLon, spec.bearing, float64(spec.pickupM)/1000)
		out[i] = catalogRide{
			id:           spec.id,
			startAddress: spec.start,
			endAddress:   spec.end,
			after:        spec.after,
			tripM:        spec.tripM,
			startLat:     lat,
			startLon:     lon,
		}
	}
	return out
}

func destination(lat, lon, bearingDeg, distKm float64) (float64, float64) {
	const earth = 6371.0
	angular := distKm / earth
	bearing := bearingDeg * math.Pi / 180
	lat1 := lat * math.Pi / 180
	lon1 := lon * math.Pi / 180
	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angular) + math.Cos(lat1)*math.Sin(angular)*math.Cos(bearing))
	lon2 := lon1 + math.Atan2(
		math.Sin(bearing)*math.Sin(angular)*math.Cos(lat1),
		math.Cos(angular)-math.Sin(lat1)*math.Sin(lat2),
	)
	return lat2 * 180 / math.Pi, lon2 * 180 / math.Pi
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earth = 6371.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	lat1r, lat2r := rad(lat1), rad(lat2)
	dLat := rad(lat2 - lat1)
	dLon := rad(lon2 - lon1)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1r)*math.Cos(lat2r)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earth * math.Asin(math.Min(1, math.Sqrt(h)))
}

func formatMeters(meters uint64) string {
	whole := meters / 1000
	frac := meters % 1000
	if frac == 0 {
		return strconv.FormatUint(whole, 10)
	}
	return strings.TrimRight(fmt.Sprintf("%d.%03d", whole, frac), "0")
}

func formatKm(km float64) string {
	if math.IsNaN(km) || math.IsInf(km, 0) || km < 0 {
		return "0"
	}
	return formatMeters(uint64(math.Round(km * 1000)))
}

func DriveCount() int {
	driveMu.Lock()
	defer driveMu.Unlock()
	return len(driveOffers)
}

func driveAt(index int) (driveOffer, error) {
	driveMu.Lock()
	defer driveMu.Unlock()
	if index < 0 || index >= len(driveOffers) {
		return driveOffer{}, errors.New("unknown ride")
	}
	return driveOffers[index], nil
}

func DriveID(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func DriveServer(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return row.Server, nil
}

func DriveCoin(index int) (int, error) {
	row, err := driveAt(index)
	if err != nil {
		return 0, err
	}
	return row.Coin, nil
}

func DriveStartAddress(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return row.StartAddress, nil
}

func DriveEndAddress(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return row.EndAddress, nil
}

func DrivePickupKm(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return formatKm(row.PickupKm), nil
}

func DriveDistanceKm(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return formatMeters(row.TripM), nil
}

func DrivePickupTime(index int) (string, error) {
	return DriveWhenClock(index)
}

func DriveImmediate(index int) (int, error) {
	row, err := driveAt(index)
	if err != nil {
		return 0, err
	}
	if row.Immediate {
		return 1, nil
	}
	return 0, nil
}

func DriveWhenDay(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return whenDay(row.When, time.Now()), nil
}

func DriveWhenClock(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return row.When.Format("15:04"), nil
}

func DriveStartLat(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatFloat(row.StartLat, 'f', 6, 64), nil
}

func DriveStartLon(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatFloat(row.StartLon, 'f', 6, 64), nil
}

func whenDay(t, now time.Time) string {
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	if ty == ny && tm == nm && td == nd {
		return ""
	}
	return t.Format("Mon")
}

func DriveFareLovelace(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(row.Fare, 10), nil
}

func DriveBaseLovelace(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(row.Base, 10), nil
}

func DrivePerKmLovelace(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(row.PerKm, 10), nil
}
