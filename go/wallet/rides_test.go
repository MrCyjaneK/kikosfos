package wallet

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFareLovelace(t *testing.T) {
	if got := kmToMeters(3.01); got != 3010 {
		t.Fatalf("meters = %d", got)
	}
	if got := fareAtoms(2000000, 500000, 3010, 1000000); got != 3505000 {
		t.Fatalf("fare = %d", got)
	}
	if got := fareAtoms(1000000, 1, 1100, 1000000); got != 1000002 {
		t.Fatalf("ceil fare = %d", got)
	}
	if got := fareAtoms(1, 1, 1, 1000000); got != 1000000 {
		t.Fatalf("floor fare = %d", got)
	}
	if got := fareAtoms(1, 1, 1, 1); got != 2 {
		t.Fatalf("satoshi fare = %d", got)
	}
	if got := fareAtoms(0, 0, 0, 1); got != 1 {
		t.Fatalf("satoshi floor = %d", got)
	}
}

func TestRegistrationFailure(t *testing.T) {
	line, ok := registrationFailure("example.com", []byte(`{"error":"driver rejected"}`))
	if !ok || line != "registration failed with example.com: driver rejected" {
		t.Fatalf("line = %q %v", line, ok)
	}
	if _, ok := registrationFailure("example.com", []byte(`{}`)); ok {
		t.Fatal("empty object")
	}
	if _, ok := registrationFailure("example.com", []byte(`{"error":""}`)); ok {
		t.Fatal("blank error")
	}
}

func TestRegisterDriversSkipAndList(t *testing.T) {
	setupTest(t)
	seen := &rideQuery{}
	inner := kikoHandler(availableWithin(munichRides, seen))
	serveKiko(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/driver" {
			seen.addPost()
		}
		inner.ServeHTTP(w, r)
	}))
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("ready.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("bare.example.com"); err != nil {
		t.Fatal(err)
	}
	msg, err := RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("without wallet: %q %v", msg, err)
	}
	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	msg, err = RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("incomplete car: %q %v", msg, err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	msg, err = RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("missing fees: %q %v", msg, err)
	}
	if err := SaveServerFees(0, "2", "0.5", "ada"); err != nil {
		t.Fatal(err)
	}
	if err := SetGPSDriver("mock"); err != nil {
		t.Fatal(err)
	}
	msg, err = RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("ready: %q %v", msg, err)
	}
	if err := RefreshDrives(5); err != nil {
		t.Fatal(err)
	}
	if n := DriveCount(); n != 4 {
		t.Fatalf("within 5 km = %d", n)
	}
	if err := RefreshDrives(0); err != nil {
		t.Fatal(err)
	}
	if n := DriveCount(); n != 4 {
		t.Fatalf("default range = %d", n)
	}
	if got := seen.rangeKm(); got != "5" {
		t.Fatalf("default range query = %s", got)
	}
	if err := RefreshDrives(8); err != nil {
		t.Fatal(err)
	}
	if got := seen.rangeKm(); got != "8" {
		t.Fatalf("range query = %s", got)
	}
	if n := DriveCount(); n != 5 {
		t.Fatalf("within 8 km = %d", n)
	}
	// The 7 km pickup is for now, so it sits with the other now rides,
	// ahead of the nearer ride that is scheduled for later.
	id, err := DriveID(2)
	nowFlag, err2 := DriveImmediate(2)
	laterFlag, err3 := DriveImmediate(3)
	if err != nil || err2 != nil || err3 != nil || id != "9f2c05" || nowFlag != 1 || laterFlag != 0 {
		t.Fatalf("now group = %s %d %d %v %v %v", id, nowFlag, laterFlag, err, err2, err3)
	}
	if err := RefreshDrives(20); err != nil {
		t.Fatal(err)
	}
	if n := DriveCount(); n != 6 {
		t.Fatalf("within 20 km = %d", n)
	}
	if err := RefreshDrives(5); err != nil {
		t.Fatal(err)
	}
	id, err = DriveID(0)
	if err != nil || id != "9f2c01" {
		t.Fatalf("nearest = %s %v", id, err)
	}
	pickup, err := DrivePickupKm(0)
	if err != nil || pickup != "0.4" {
		t.Fatalf("pickup = %s %v", pickup, err)
	}
	var prev float64
	for i := 0; i < DriveCount(); i++ {
		km, err := DrivePickupKm(i)
		if err != nil {
			t.Fatal(err)
		}
		got := parseKm(t, km)
		if got < prev {
			t.Fatalf("order %d: %s", i, km)
		}
		prev = got
		host, err := DriveServer(i)
		if err != nil || host != "ready.example.com" {
			t.Fatalf("server %d = %s %v", i, host, err)
		}
		flag, err := DriveImmediate(i)
		if err != nil {
			t.Fatal(err)
		}
		wantNow := 0
		if i < 2 {
			wantNow = 1
		}
		if flag != wantNow {
			t.Fatalf("immediate %d = %d", i, flag)
		}
		clock, err := DriveWhenClock(i)
		if err != nil || len(clock) != 5 || clock[2] != ':' {
			t.Fatalf("clock %d = %q %v", i, clock, err)
		}
	}
	day, err := DriveWhenDay(3)
	if err != nil || day == "" {
		t.Fatalf("later day = %q %v", day, err)
	}
	if _, err := DriveID(99); err == nil || err.Error() != "unknown ride" {
		t.Fatalf("unknown ride: %v", err)
	}
	if seen.posts() != 1 {
		t.Fatalf("driver posts = %d", seen.posts())
	}
	material, err := WalletMultisigMaterial(0)
	if err != nil || seen.key() != material {
		t.Fatalf("driver key = %s %v", seen.key(), err)
	}
	pos, err := GPSPosition()
	if err != nil {
		t.Fatal(err)
	}
	if !sameFix(t, seen.location(), pos) {
		t.Fatalf("location = %s want %s", seen.location(), pos)
	}
}

func TestRegisterDriversSplitFares(t *testing.T) {
	setupTest(t)
	serveKikoHosts(t, map[string]http.Handler{
		"a.example.com": kikoHandler(availableFixed(aRides)),
		"b.example.com": kikoHandler(availableFixed(bRides)),
	})
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if err := SetGPSDriver("mock"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("a.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("b.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(0, "2", "0.5", "ada"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(1, "3", "1", "ada"); err != nil {
		t.Fatal(err)
	}
	msg, err := RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("register: %q %v", msg, err)
	}
	if err := RefreshDrives(5); err != nil {
		t.Fatal(err)
	}
	if n := DriveCount(); n != 4 {
		t.Fatalf("count = %d", n)
	}
	want := []struct {
		server, base, perKm, fare, distance string
	}{
		{"a.example.com", "2000000", "500000", "2800000", "1.6"},
		{"b.example.com", "3000000", "1000000", "6000000", "3"},
		{"a.example.com", "2000000", "500000", "4100000", "4.2"},
		{"b.example.com", "3000000", "1000000", "5100000", "2.1"},
	}
	for i, row := range want {
		server, err := DriveServer(i)
		base, err2 := DriveBaseLovelace(i)
		perKm, err3 := DrivePerKmLovelace(i)
		fare, err4 := DriveFareLovelace(i)
		distance, err5 := DriveDistanceKm(i)
		if err != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			t.Fatalf("row %d: %v %v %v %v %v", i, err, err2, err3, err4, err5)
		}
		if server != row.server || base != row.base || perKm != row.perKm || fare != row.fare || distance != row.distance {
			t.Fatalf("row %d = %s %s %s %s %s", i, server, base, perKm, fare, distance)
		}
		coin, err := DriveCoin(i)
		if err != nil || coin != 0 {
			t.Fatalf("coin %d = %d %v", i, coin, err)
		}
	}
}

func TestRegisterDriversServerError(t *testing.T) {
	setupTest(t)
	serveKiko(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/currency" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"currency":"cardano"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"driver rejected"}`))
	}))
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if err := SetGPSDriver("mock"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("bad.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("worse.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(0, "2", "0.5", "ada"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(1, "3", "1", "ada"); err != nil {
		t.Fatal(err)
	}
	msg, err := RegisterDrivers()
	if err != nil {
		t.Fatal(err)
	}
	want := "registration failed with bad.example.com: driver rejected\nregistration failed with worse.example.com: driver rejected"
	if msg != want {
		t.Fatalf("msg = %q", msg)
	}
	if err := RefreshDrives(5); err != nil {
		t.Fatal(err)
	}
	if n := DriveCount(); n != 0 {
		t.Fatalf("rejected servers still listed %d", n)
	}
}

func TestClaimAndVerify(t *testing.T) {
	setupTest(t)
	var etaMin int
	var etaKm float64
	var gotKey, gotPath string
	serveKiko(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/currency" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"currency":"cardano"}`))
		case r.URL.Path == "/api/v1/driver" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/driver/drive/taken":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"ride is no longer available"}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/driver/drive/"):
			gotPath = r.URL.Path
			gotKey = r.Header.Get("X-Driver-Key")
			var body struct {
				EtaMinutes int     `json:"eta_minutes"`
				EtaKm      float64 `json:"eta_km"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			etaMin = body.EtaMinutes
			etaKm = body.EtaKm
			_, _ = w.Write([]byte(`{"status":"accepted","pay_to":"addr_test1qx","fare":2800000,"start_address":"Prielmayerstraße","end_address":"Karlsplatz","start":{"lat":48.140833,"lng":11.555},"end":{"lat":48.150833,"lng":11.565},"distance_km":1.6,"pickup_time":"2026-11-01T09:00:00Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("ready.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(0, "2", "0.5", "ada"); err != nil {
		t.Fatal(err)
	}
	if msg, err := RegisterDrivers(); err != nil || msg != "" {
		t.Fatalf("register: %q %v", msg, err)
	}
	if _, err := VerifyDrive("ready.example.com", "9f2c01"); err == nil || err.Error() != "claim the ride first" {
		t.Fatalf("before claim: %v", err)
	}
	if err := ClaimDrive("ready.example.com", "taken", "1.2"); err == nil || !strings.Contains(err.Error(), "no longer available") {
		t.Fatalf("taken: %v", err)
	}
	if DriveClaimed("ready.example.com", "taken") != 0 || ActiveCount() != 0 {
		t.Fatal("rejected claim was stored")
	}
	if err := ClaimDrive("ready.example.com", "9f2c01", "0.4"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/v1/driver/drive/9f2c01" || etaMin != 1 || math.Abs(etaKm-0.4) > 1e-9 {
		t.Fatalf("accept = %s %d %v", gotPath, etaMin, etaKm)
	}
	material, err := WalletMultisigMaterial(0)
	if err != nil || gotKey != material {
		t.Fatalf("key = %s %v", gotKey, err)
	}
	if DriveClaimed("Ready.example.com", "9f2c01") != 1 || DrivePaid("ready.example.com", "9f2c01") != 0 {
		t.Fatal("claimed state")
	}
	if ActiveCount() != 1 {
		t.Fatalf("active = %d", ActiveCount())
	}
	raw, err := ActiveRide(0)
	if err != nil {
		t.Fatal(err)
	}
	var shown struct {
		ID           string `json:"id"`
		StartAddress string `json:"start_address"`
		EndAddress   string `json:"end_address"`
		PickupKm     string `json:"pickup_km"`
		DistanceKm   string `json:"distance_km"`
		Fare         string `json:"fare_lovelace"`
		StartLat     string `json:"start_lat"`
		EndLat       string `json:"end_lat"`
	}
	if err := json.Unmarshal([]byte(raw), &shown); err != nil {
		t.Fatal(err)
	}
	if shown.ID != "9f2c01" || shown.StartAddress != "Prielmayerstraße" || shown.EndAddress != "Karlsplatz" || shown.PickupKm != "0.4" || shown.DistanceKm != "1.6" || shown.Fare != "2800000" || shown.StartLat != "48.140833" || shown.EndLat != "48.150833" {
		t.Fatalf("active = %+v", shown)
	}
	if _, err := ActiveRide(1); err == nil {
		t.Fatal("expected no second active ride")
	}
	prev := paymentSeen
	t.Cleanup(func() { paymentSeen = prev })
	var addr, amount string
	paymentSeen = func(c Coin, address, atoms string) (bool, error) {
		if c.ID() != "cardano" {
			t.Fatalf("coin = %s", c.ID())
		}
		addr, amount = address, atoms
		return false, nil
	}
	paid, err := VerifyDrive("ready.example.com", "9f2c01")
	if err != nil || paid || addr != "addr_test1qx" || amount != "2800000" {
		t.Fatalf("unpaid = %v %v %s %s", paid, err, addr, amount)
	}
	paymentSeen = func(Coin, string, string) (bool, error) { return true, nil }
	paid, err = VerifyDrive("ready.example.com", "9f2c01")
	if err != nil || !paid || DrivePaid("ready.example.com", "9f2c01") != 1 {
		t.Fatalf("paid = %v %v", paid, err)
	}
	calls := 0
	paymentSeen = func(Coin, string, string) (bool, error) {
		calls++
		return false, nil
	}
	paid, err = VerifyDrive("ready.example.com", "9f2c01")
	if err != nil || !paid || calls != 0 {
		t.Fatalf("remembered = %v %v calls %d", paid, err, calls)
	}
}

func TestCashOutAndReload(t *testing.T) {
	setupTest(t)
	var payoutBody string
	var payoutKey string
	serveKiko(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/currency" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"currency":"cardano"}`))
		case r.URL.Path == "/api/v1/driver" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/driver/drive/9f2c01":
			_, _ = w.Write([]byte(`{"status":"accepted","pay_to":"addr1qy","fare_lovelace":2800000,"start_address":"Prielmayerstraße","end_address":"Karlsplatz","start":{"lat":48.14,"lng":11.55},"end":{"lat":48.15,"lng":11.56},"distance_km":1.6,"pickup_time":"2026-11-01T09:00:00Z","address_index":4,"script":"aabb"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/driver/drive/9f2c01":
			_, _ = w.Write([]byte(`{"status":"paid","pay_to":"addr1qy","fare_lovelace":2800000,"start_address":"Prielmayerstraße","end_address":"Karlsplatz","start":{"lat":48.14,"lng":11.55},"end":{"lat":48.15,"lng":11.56},"distance_km":1.6,"pickup_time":"2026-11-01T09:00:00Z","eta_km":0.4,"address_index":4,"script":"aabb"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/driver/drive/9f2c01/payout":
			payoutKey = r.Header.Get("X-Driver-Key")
			raw, _ := io.ReadAll(r.Body)
			payoutBody = string(raw)
			_, _ = w.Write([]byte(`{"status":"paid_out"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("ready.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(0, "2", "0.5", "ada"); err != nil {
		t.Fatal(err)
	}
	if msg, err := RegisterDrivers(); err != nil || msg != "" {
		t.Fatalf("register: %q %v", msg, err)
	}
	if err := ClaimDrive("ready.example.com", "9f2c01", "0.4"); err != nil {
		t.Fatal(err)
	}
	if err := CashOutDrive("ready.example.com", "9f2c01"); err == nil || err.Error() != "the ride is not paid" {
		t.Fatalf("unpaid cash out: %v", err)
	}
	prevPay := paymentSeen
	prevBuild := buildPayout
	t.Cleanup(func() {
		paymentSeen = prevPay
		buildPayout = prevBuild
	})
	paymentSeen = func(Coin, string, string) (bool, error) { return true, nil }
	if paid, err := VerifyDrive("ready.example.com", "9f2c01"); err != nil || !paid {
		t.Fatal(err)
	}
	buildPayout = func(mnemonic, payTo, script string, index uint32) (string, error) {
		if payTo != "addr1qy" || script != "aabb" || index != 4 || mnemonic == "" {
			t.Fatalf("payout args %q %q %d", payTo, script, index)
		}
		return "abcd", nil
	}
	if err := CashOutDrive("ready.example.com", "9f2c01"); err != nil {
		t.Fatal(err)
	}
	material, err := WalletMultisigMaterial(0)
	if err != nil || payoutKey != material || payoutBody != `{"transaction":"abcd"}` {
		t.Fatalf("posted %s %s", payoutKey, payoutBody)
	}
	if ActiveCount() != 0 {
		t.Fatal("cashed-out ride is still active")
	}

	if err := ClaimDrive("ready.example.com", "9f2c01", "0.4"); err != nil {
		t.Fatal(err)
	}
	clearHeld()
	if ActiveCount() != 0 {
		t.Fatal("clear did not empty the list")
	}
	if msg, err := RegisterDrivers(); err != nil || msg != "" {
		t.Fatalf("reload register: %q %v", msg, err)
	}
	if ActiveCount() != 1 || DrivePaid("ready.example.com", "9f2c01") != 1 {
		t.Fatalf("reloaded count %d paid %d", ActiveCount(), DrivePaid("ready.example.com", "9f2c01"))
	}
	raw, err := ActiveRide(0)
	if err != nil {
		t.Fatal(err)
	}
	var shown struct {
		Status   string `json:"status"`
		PickupKm string `json:"pickup_km"`
		Paid     int    `json:"paid"`
	}
	if err := json.Unmarshal([]byte(raw), &shown); err != nil {
		t.Fatal(err)
	}
	if shown.Status != "paid" || shown.Paid != 1 || shown.PickupKm != "0.4" {
		t.Fatalf("reloaded = %+v", shown)
	}
}

type sampleRide struct {
	id, start, end string
	after          time.Duration
	pickup, trip   float64
	lat, lng       float64
}

func (ride sampleRide) payload(now time.Time) remoteRide {
	row := remoteRide{
		ID:               ride.id,
		StartAddress:     ride.start,
		EndAddress:       ride.end,
		DistanceKm:       ride.trip,
		PickupDistanceKm: ride.pickup,
		PickupTime:       now.Add(ride.after).UTC().Format(time.RFC3339),
	}
	row.Start.Lat = ride.lat
	row.Start.Lng = ride.lng
	row.End.Lat = ride.lat + 0.01
	row.End.Lng = ride.lng + 0.01
	return row
}

var munichRides = []sampleRide{
	{"9f2c01", "Prielmayerstraße", "Karlsplatz", 5 * time.Minute, 0.4, 1.6, 48.140833, 11.555},
	{"9f2c02", "Stachus", "Odeonsplatz", 12 * time.Minute, 1.2, 3, 48.15, 11.57},
	{"9f2c03", "Sendlinger Tor", "Münchner Freiheit", 3 * time.Hour, 2.5, 4.2, 48.13, 11.56},
	{"9f2c04", "Theresienwiese", "Hauptbahnhof", 26 * time.Hour, 4.8, 2.1, 48.12, 11.54},
	{"9f2c05", "Nymphenburg", "Marienplatz", 8 * time.Minute, 7, 8, 48.16, 11.5},
	{"9f2c06", "Garching", "Hauptbahnhof", 50 * time.Hour, 12, 15, 48.25, 11.65},
}

var aRides = []sampleRide{munichRides[0], munichRides[2]}
var bRides = []sampleRide{munichRides[1], munichRides[3]}

type rideQuery struct {
	mu       sync.Mutex
	rangeTxt string
	loc      string
	driver   string
	postN    int
}

func (q *rideQuery) note(r *http.Request) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rangeTxt = r.URL.Query().Get("range")
	q.loc = r.URL.Query().Get("location")
	q.driver = r.Header.Get("X-Driver-Key")
}

func (q *rideQuery) addPost() {
	q.mu.Lock()
	q.postN++
	q.mu.Unlock()
}

func (q *rideQuery) rangeKm() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.rangeTxt
}

func (q *rideQuery) location() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.loc
}

func (q *rideQuery) key() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.driver
}

func (q *rideQuery) posts() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.postN
}

func availableWithin(catalog []sampleRide, seen *rideQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Driver-Key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"missing key"}`))
			return
		}
		rangeKm, err := strconv.ParseFloat(r.URL.Query().Get("range"), 64)
		if err != nil || rangeKm <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"range must be a positive number of km"}`))
			return
		}
		if seen != nil {
			seen.note(r)
		}
		now := time.Now()
		out := []remoteRide{}
		for _, ride := range catalog {
			if ride.pickup <= rangeKm {
				out = append(out, ride.payload(now))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func availableFixed(catalog []sampleRide) http.HandlerFunc {
	return availableWithin(catalog, nil)
}

func sameFix(t *testing.T, location, position string) bool {
	t.Helper()
	halves := strings.Split(location, ",")
	if len(halves) != 2 {
		return false
	}
	lines := strings.Split(position, "\n")
	if len(lines) < 2 {
		return false
	}
	lat, err := strconv.ParseFloat(halves[0], 64)
	lon, err2 := strconv.ParseFloat(halves[1], 64)
	wantLat, err3 := strconv.ParseFloat(lines[0], 64)
	wantLon, err4 := strconv.ParseFloat(lines[1], 64)
	if err != nil || err2 != nil || err3 != nil || err4 != nil {
		return false
	}
	return math.Abs(lat-wantLat) < 1e-5 && math.Abs(lon-wantLon) < 1e-5
}

func parseKm(t *testing.T, text string) float64 {
	t.Helper()
	got, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(got) {
		t.Fatalf("km %q: %v", text, err)
	}
	return got
}
