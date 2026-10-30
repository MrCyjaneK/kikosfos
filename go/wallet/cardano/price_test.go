package cardano

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFiatQuotesCachedBalance(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	eurRateMu.Lock()
	eurRate.price = nil
	eurRate.until = time.Time{}
	eurRateMu.Unlock()

	var priceHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			priceHits++
			_, _ = w.Write([]byte(`{"results":{"100":0.5,"200":0.25}}`))
			return
		}
		_, _ = w.Write([]byte(`[{"balance":"2000000"}]`))
	}))
	defer srv.Close()

	prevRate := eurRateEndpoint
	prevBalance := balanceEndpoint
	prevClient := balanceClient
	eurRateEndpoint = srv.URL
	balanceEndpoint = srv.URL
	balanceClient = srv.Client()
	t.Cleanup(func() {
		eurRateEndpoint = prevRate
		balanceEndpoint = prevBalance
		balanceClient = prevClient
		eurRateMu.Lock()
		eurRate.price = nil
		eurRate.until = time.Time{}
		eurRateMu.Unlock()
	})

	ada, err := Balance("addr1eur")
	if err != nil {
		t.Fatal(err)
	}
	if ada != "2 ADA" {
		t.Fatalf("ada = %s", ada)
	}
	got, err := Fiat("addr1eur")
	if err != nil {
		t.Fatal(err)
	}
	if got != "€0.50" {
		t.Fatalf("eur = %s", got)
	}
	if _, err := Fiat("addr1eur"); err != nil {
		t.Fatal(err)
	}
	if priceHits != 1 {
		t.Fatalf("price requests = %d", priceHits)
	}
}

func TestFormatEURSmallAmount(t *testing.T) {
	price := new(big.Rat)
	if _, ok := price.SetString("0.10"); !ok {
		t.Fatal("bad price")
	}
	got, err := formatEUR("1", price)
	if err != nil {
		t.Fatal(err)
	}
	if got != "< €0.01" {
		t.Fatalf("eur = %s", got)
	}
}

func TestEurRateURLUsesTicker(t *testing.T) {
	prev := eurRateEndpoint
	eurRateEndpoint = ""
	t.Cleanup(func() { eurRateEndpoint = prev })
	if Ticker != "ADA" {
		t.Fatal(Ticker)
	}
	const want = "https://prices.cakewallet.com/v3/rates?base=crypto.ADA&quote=fiat.EUR"
	if eurRateURL() != want {
		t.Fatalf("url = %s", eurRateURL())
	}
}

func TestAmountFromEURUsesCachedPrice(t *testing.T) {
	setRate := func(price *big.Rat) {
		eurRateMu.Lock()
		eurRate.price = new(big.Rat).Set(price)
		eurRate.until = time.Now().Add(time.Minute)
		eurRateMu.Unlock()
	}
	setRate(big.NewRat(1, 4))
	t.Cleanup(func() {
		eurRateMu.Lock()
		eurRate.price = nil
		eurRate.until = time.Time{}
		eurRateMu.Unlock()
	})

	got, err := AmountFromEUR("0.50")
	if err != nil || got != "2000000" {
		t.Fatalf("0.50 = %s %v", got, err)
	}
	got, err = AmountFromEUR("0,50")
	if err != nil || got != "2000000" {
		t.Fatalf("0,50 = %s %v", got, err)
	}
	got, err = AmountFromEUR("€ 1")
	if err != nil || got != "4000000" {
		t.Fatalf("1 = %s %v", got, err)
	}
	got, err = AmountFromEUR("0")
	if err != nil || got != "0" {
		t.Fatalf("0 = %s %v", got, err)
	}
	if _, err := AmountFromEUR(""); err == nil {
		t.Fatal("expected empty euro to fail")
	}
	if _, err := AmountFromEUR("-1"); err == nil {
		t.Fatal("expected negative euro to fail")
	}

	setRate(big.NewRat(3, 10))
	got, err = AmountFromEUR("1")
	if err != nil || got != "3333333" {
		t.Fatalf("rounded = %s %v", got, err)
	}

	setRate(big.NewRat(1, 1))
	got, err = AmountFromEUR("0.0000005")
	if err != nil || got != "1" {
		t.Fatalf("half lovelace = %s %v", got, err)
	}
}
