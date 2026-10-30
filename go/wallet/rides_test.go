package wallet

import (
	"math"
	"strconv"
	"testing"
)

func TestFareLovelace(t *testing.T) {
	if got := fareLovelace(2000000, 500000, 3010); got != 3505000 {
		t.Fatalf("fare = %d", got)
	}
	if got := fareLovelace(1000000, 1, 1100); got != 1000002 {
		t.Fatalf("ceil fare = %d", got)
	}
	if got := fareLovelace(1, 1, 1); got != 1000000 {
		t.Fatalf("floor fare = %d", got)
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
	if err := RefreshDrives(8); err != nil {
		t.Fatal(err)
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
}

func TestRegisterDriversSplitFares(t *testing.T) {
	setupTest(t)
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
	prev := postDriver
	postDriver = func([]byte) ([]byte, error) {
		return []byte(`{"error":"driver rejected"}`), nil
	}
	t.Cleanup(func() { postDriver = prev })
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

func parseKm(t *testing.T, text string) float64 {
	t.Helper()
	got, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(got) {
		t.Fatalf("km %q: %v", text, err)
	}
	return got
}
