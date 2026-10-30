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
