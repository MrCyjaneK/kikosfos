package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"net/url"
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
	EndLat       float64
	EndLon       float64
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
	Key   string
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
	clearHeld()
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
	key, err := WalletMultisigMaterial(coin)
	if err != nil {
		return readyServer{}, err
	}
	return readyServer{Host: host, Coin: coin, Base: base, PerKm: perKm, Key: key}, nil
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

// RefreshDrives loads requested rides from each registered server.
// rangeKm <= 0 means 5. The server keeps pickups inside that radius.
// The list is empty when no server registered. No fix, or a server that
// does not answer, clears the list and returns that error.
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
	rows, err := offersFrom(readyCopy(), lat, lon, rangeKm)
	if err != nil {
		setOffers(nil)
		return err
	}
	setOffers(rows)
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

// remoteRide is one row of GET /api/v1/driver/drive/available.
type remoteRide struct {
	ID           string `json:"id"`
	StartAddress string `json:"start_address"`
	EndAddress   string `json:"end_address"`
	Start        struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"start"`
	End struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"end"`
	DistanceKm       float64 `json:"distance_km"`
	PickupDistanceKm float64 `json:"pickup_distance_km"`
	PickupTime       string  `json:"pickup_time"`
}

func offersFrom(ready []readyServer, lat, lon, rangeKm float64) ([]driveOffer, error) {
	if len(ready) == 0 {
		return nil, nil
	}
	now := time.Now()
	var rows []driveOffer
	for _, server := range ready {
		part, err := ridesFrom(server, lat, lon, rangeKm, now)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
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
	return rows, nil
}

func ridesFrom(server readyServer, lat, lon, rangeKm float64, now time.Time) ([]driveOffer, error) {
	query := url.Values{}
	query.Set("location", strconv.FormatFloat(lat, 'f', -1, 64)+","+strconv.FormatFloat(lon, 'f', -1, 64))
	query.Set("range", strconv.FormatFloat(rangeKm, 'f', -1, 64))
	status, body, err := kikoDo(http.MethodGet, server.Host, "/api/v1/driver/drive/available?"+query.Encode(), nil, map[string]string{
		"X-Driver-Key": server.Key,
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, errors.New(server.Host + ": " + apiError(status, body))
	}
	var listed []remoteRide
	if json.Unmarshal(body, &listed) != nil {
		return nil, errors.New(server.Host + ": bad ride list")
	}
	rows := make([]driveOffer, 0, len(listed))
	for _, ride := range listed {
		when, err := time.Parse(time.RFC3339, ride.PickupTime)
		if err != nil {
			when, err = time.Parse(time.RFC3339Nano, ride.PickupTime)
		}
		if err != nil {
			return nil, errors.New(server.Host + ": bad pickup time")
		}
		meters := kmToMeters(ride.DistanceKm)
		rows = append(rows, driveOffer{
			ID:           ride.ID,
			Server:       server.Host,
			Coin:         server.Coin,
			StartAddress: ride.StartAddress,
			EndAddress:   ride.EndAddress,
			PickupKm:     ride.PickupDistanceKm,
			TripM:        meters,
			When:         when,
			StartLat:     ride.Start.Lat,
			StartLon:     ride.Start.Lng,
			EndLat:       ride.End.Lat,
			EndLon:       ride.End.Lng,
			Immediate:    !when.After(now.Add(nowWindow)),
			Fare:         fareLovelace(server.Base, server.PerKm, meters),
			Base:         server.Base,
			PerKm:        server.PerKm,
		})
	}
	return rows, nil
}

func kmToMeters(km float64) uint64 {
	if math.IsNaN(km) || math.IsInf(km, 0) || km <= 0 {
		return 0
	}
	meters := math.Round(km * 1000)
	if meters >= math.MaxUint64 {
		return math.MaxUint64
	}
	return uint64(meters)
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

func DriveEndLat(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatFloat(row.EndLat, 'f', 6, 64), nil
}

func DriveEndLon(index int) (string, error) {
	row, err := driveAt(index)
	if err != nil {
		return "", err
	}
	return strconv.FormatFloat(row.EndLon, 'f', 6, 64), nil
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
