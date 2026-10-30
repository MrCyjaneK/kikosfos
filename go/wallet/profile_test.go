package wallet

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGPSDriver(t *testing.T) {
	setupTest(t)
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	kind, err := GPSDriver()
	if err != nil || kind != "geoclue" {
		t.Fatalf("driver %q %v", kind, err)
	}
	if err := SetGPSDriver("mock"); err != nil {
		t.Fatal(err)
	}
	kind, err = GPSDriver()
	if err != nil || kind != "mock" {
		t.Fatalf("driver %q %v", kind, err)
	}
	text, err := GPSPosition()
	if err != nil {
		t.Fatal(err)
	}
	if text != "48.140833\n11.555000\n15" {
		t.Fatalf("position %q", text)
	}
	if err := SetGPSDriver("nope"); err != nil {
		t.Fatal(err)
	}
	kind, err = GPSDriver()
	if err != nil || kind != "geoclue" {
		t.Fatalf("driver %q %v", kind, err)
	}
}

func TestProfileRoundTrip(t *testing.T) {
	setupTest(t)
	if _, err := ProfileBrand(); err == nil {
		t.Fatal("expected locked")
	}
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	brand, err := ProfileBrand()
	if err != nil || brand != "" {
		t.Fatalf("brand = %q %v", brand, err)
	}
	if err := SaveCar(" VW ", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	car, err := ProfileCar()
	if err != nil {
		t.Fatal(err)
	}
	if car.Brand != "VW" || car.Model != "ID.3" || car.Color != "white" || car.LicensePlate != "B-KK 402" || car.Year != 2024 {
		t.Fatalf("car = %+v", car)
	}
	year, err := ProfileYear()
	if err != nil || year != "2024" {
		t.Fatalf("year = %q %v", year, err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "nope"); err == nil {
		t.Fatal("expected bad year")
	}
	car, err = ProfileCar()
	if err != nil || car.Year != 2024 {
		t.Fatalf("year changed on bad save %+v %v", car, err)
	}

	storeMu.RLock()
	path := filepath.Join(currentStore.ConfigDir, "profile")
	storeMu.RUnlock()
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := append([]byte(nil), blob...)
	header[4] ^= 0x01
	if err := os.WriteFile(path, header, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ProfileBrand(); err == nil {
		t.Fatal("expected tampered header to fail")
	}
	body := append([]byte(nil), blob...)
	body[len(body)-1] ^= 0x01
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ProfileBrand(); err == nil {
		t.Fatal("expected tampered profile to fail")
	}
}

func TestServersFeesAndDriver(t *testing.T) {
	setupTest(t)
	serveKiko(t, nil)
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer(" https://example.com "); err == nil {
		t.Fatal("expected scheme to be rejected")
	}
	if err := AddServer("example.com/api"); err == nil {
		t.Fatal("expected path to be rejected")
	}
	if err := AddServer(""); err == nil {
		t.Fatal("expected empty host to be rejected")
	}
	if err := AddServer("example.com"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("Example.com"); err == nil {
		t.Fatal("expected duplicate host")
	}
	n, err := ServerCount()
	if err != nil || n != 1 {
		t.Fatalf("count = %d %v", n, err)
	}
	currency, err := ServerCurrency(0)
	if err != nil || currency != "cardano" {
		t.Fatalf("currency = %s %v", currency, err)
	}
	ok, err := ServerCompatible(0)
	if err != nil || ok {
		t.Fatalf("compatible without wallet = %v %v", ok, err)
	}
	name, err := ServerWalletName(0)
	if err != nil || name != "" {
		t.Fatalf("name = %q %v", name, err)
	}
	if _, err := RegisterDriver("example.com"); !errors.Is(err, errIncompatible) {
		t.Fatalf("register without wallet: %v", err)
	}

	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	if registry[0].Ticker() != "ADA" {
		t.Fatal(registry[0].Ticker())
	}
	ok, err = ServerCompatible(0)
	if err != nil || !ok {
		t.Fatalf("compatible = %v %v", ok, err)
	}
	name, err = ServerWalletName(0)
	if err != nil || name != "Cardano" {
		t.Fatalf("name = %q %v", name, err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterDriver("example.com"); err == nil || err.Error() != "fees are not set" {
		t.Fatalf("register without fees: %v", err)
	}

	orig := feeFromEUR
	t.Cleanup(func() { feeFromEUR = orig })
	feeFromEUR = func(Coin, string) (string, error) {
		return "", errors.New("no price")
	}
	if err := ApplyFeesEUR("0.50", "0.25"); err == nil || err.Error() != "no price" {
		t.Fatalf("apply price error: %v", err)
	}
	base, err := ServerFeeBase(0)
	if err != nil || base != "" {
		t.Fatalf("fee after failed apply = %q %v", base, err)
	}

	feeFromEUR = func(c Coin, eur string) (string, error) {
		if c.Ticker() != "ADA" {
			t.Fatalf("ticker = %s", c.Ticker())
		}
		switch eur {
		case "0.50":
			return "2000000", nil
		case "0.25":
			return "1000000", nil
		default:
			return "", errors.New("unexpected eur " + eur)
		}
	}
	if err := ApplyFeesEUR("0.50", "0.25"); err != nil {
		t.Fatal(err)
	}
	per, err := ServerFeePerKm(0)
	base, err = ServerFeeBase(0)
	if err != nil || base != "2000000" || per != "1000000" {
		t.Fatalf("applied fees = %s %s %v", base, per, err)
	}

	if err := SaveServerFees(0, "3", "1.5", "ada"); err != nil {
		t.Fatal(err)
	}
	if err := AddServer("other.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(1, "1", "1", "ada"); err != nil {
		t.Fatal(err)
	}
	base, err = ServerFeeBase(0)
	other, err := ServerFeeBase(1)
	if err != nil || base != "3000000" || other != "1000000" {
		t.Fatalf("per-server fees = %s %s %v", base, other, err)
	}
	if err := DeleteServer(1); err != nil {
		t.Fatal(err)
	}
	per, err = ServerFeePerKm(0)
	if err != nil || per != "1500000" {
		t.Fatalf("per km = %s %v", per, err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	host, err := ServerHost(0)
	if err != nil || host != "example.com" {
		t.Fatalf("host after car save = %s %v", host, err)
	}
	base, err = ServerFeeBase(0)
	if err != nil || base != "3000000" {
		t.Fatalf("fee after car save = %s %v", base, err)
	}

	if err := UpdateServer(0, "ride.example.com"); err != nil {
		t.Fatal(err)
	}
	host, err = ServerHost(0)
	currency, err = ServerCurrency(0)
	base, err = ServerFeeBase(0)
	if err != nil || host != "ride.example.com" || currency != "cardano" || base != "3000000" {
		t.Fatalf("updated server = %s %s %s %v", host, currency, base, err)
	}

	material, err := WalletMultisigMaterial(0)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := DriverRequest("ride.example.com")
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		PublicKey string `json:"public_key"`
		BaseFee   uint64 `json:"base_fee"`
		FeePerKm  uint64 `json:"fee_per_km"`
		Car       Car    `json:"car"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.PublicKey != material || !strings.HasPrefix(body.PublicKey, "acct_shared_xvk") {
		t.Fatalf("public key = %s", body.PublicKey)
	}
	if body.BaseFee != 3000000 || body.FeePerKm != 1500000 {
		t.Fatalf("fees = %d %d", body.BaseFee, body.FeePerKm)
	}
	if body.Car.Brand != "VW" || body.Car.Model != "ID.3" || body.Car.Color != "white" || body.Car.LicensePlate != "B-KK 402" || body.Car.Year != 2024 {
		t.Fatalf("car = %+v", body.Car)
	}
	resp, err := RegisterDriver("ride.example.com")
	if err != nil {
		t.Fatal(err)
	}
	var echoed driverBody
	if err := json.Unmarshal(resp, &echoed); err != nil {
		t.Fatalf("register = %s %v", resp, err)
	}
	if echoed.PublicKey != material || echoed.BaseFee != 3000000 || echoed.FeePerKm != 1500000 || echoed.Car.Brand != "VW" {
		t.Fatalf("register = %+v", echoed)
	}

	if err := SaveCar("", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterDriver("ride.example.com"); err == nil || err.Error() != "car is incomplete" {
		t.Fatalf("incomplete car: %v", err)
	}
	if err := SaveCar("VW", "ID.3", "white", "B-KK 402", "2024"); err != nil {
		t.Fatal(err)
	}
	if err := WalletDelete(0, "42"); err != nil {
		t.Fatal(err)
	}
	base, err = ServerFeeBase(0)
	if err != nil || base != "3000000" {
		t.Fatalf("fee after wallet delete = %q %v", base, err)
	}
	if _, err := RegisterDriver("ride.example.com"); !errors.Is(err, errIncompatible) {
		t.Fatalf("register after delete: %v", err)
	}
	if err := DeleteServer(0); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterDriver("ride.example.com"); err == nil || err.Error() != "unknown server" {
		t.Fatalf("missing server: %v", err)
	}
	if err := ApplyFeesEUR("1", "1"); err == nil || err.Error() != "no servers" {
		t.Fatalf("apply without servers: %v", err)
	}
}
