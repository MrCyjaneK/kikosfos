package bsv

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func clearEuroCache() {
	eurRateMu.Lock()
	eurRate.price = nil
	eurRate.until = time.Time{}
	eurRateMu.Unlock()
}

func TestEurRateURLUsesTicker(t *testing.T) {
	prev := eurRateEndpoint
	eurRateEndpoint = ""
	t.Cleanup(func() { eurRateEndpoint = prev })
	const want = "https://prices.cakewallet.com/v3/rates?base=crypto.BSV&quote=fiat.EUR"
	if eurRateURL() != want {
		t.Fatalf("url = %s", eurRateURL())
	}
}

func TestFiatQuotesCachedBalance(t *testing.T) {
	clearBalanceCache()
	clearEuroCache()
	t.Cleanup(clearBalanceCache)
	t.Cleanup(clearEuroCache)

	var priceHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rates":
			priceHits++
			_, _ = w.Write([]byte(`{"results":{"100":0.5,"200":0.25}}`))
		case "/address/" + abandonAddress + "/balance":
			_, _ = w.Write([]byte(`{"confirmed":200000000,"unconfirmed":0}`))
		default:
			t.Errorf("path = %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	useChain(t, srv)
	eurRateEndpoint = srv.URL + "/rates"

	got, err := Balance(abandonAddress)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2 BSV" {
		t.Fatalf("bsv = %s", got)
	}
	eur, err := Fiat(abandonAddress)
	if err != nil {
		t.Fatal(err)
	}
	if eur != "€0.50" {
		t.Fatalf("eur = %s", eur)
	}
	if _, err := Fiat(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if priceHits != 1 {
		t.Fatalf("price requests = %d", priceHits)
	}
}

func TestFiatMissingRate(t *testing.T) {
	clearBalanceCache()
	clearEuroCache()
	t.Cleanup(clearBalanceCache)
	t.Cleanup(clearEuroCache)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rates" {
			_, _ = w.Write([]byte(`{"results":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"confirmed":100000000,"unconfirmed":0}`))
	}))
	defer srv.Close()
	useChain(t, srv)
	eurRateEndpoint = srv.URL + "/rates"
	if _, err := Fiat(abandonAddress); err == nil {
		t.Fatal("expected missing rate to fail")
	}
}

func TestQuoteUsesCachedPriceWithoutNetwork(t *testing.T) {
	clearEuroCache()
	t.Cleanup(clearEuroCache)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "unused", http.StatusInternalServerError)
	}))
	defer srv.Close()
	useChain(t, srv)

	text, sats, eur, err := Quote("2", Whole)
	if err != nil {
		t.Fatal(err)
	}
	if text != "2 BSV" || sats != "200000000" || eur != "" {
		t.Fatalf("quote without price = %s %s %s", text, sats, eur)
	}
	if hits != 0 {
		t.Fatalf("quote made %d requests", hits)
	}

	eurRateMu.Lock()
	eurRate.price = big.NewRat(1, 4)
	eurRate.until = time.Now().Add(time.Minute)
	eurRateMu.Unlock()
	text, sats, eur, err = Quote("2", Whole)
	if err != nil {
		t.Fatal(err)
	}
	if text != "2 BSV" || sats != "200000000" || eur != "€0.50" {
		t.Fatalf("quote with price = %s %s %s", text, sats, eur)
	}
	if hits != 0 {
		t.Fatalf("cached quote made %d requests", hits)
	}
}

func TestAmountFromEURUsesCachedPrice(t *testing.T) {
	clearEuroCache()
	t.Cleanup(clearEuroCache)
	eurRateMu.Lock()
	eurRate.price = big.NewRat(1, 4)
	eurRate.until = time.Now().Add(time.Minute)
	eurRateMu.Unlock()
	got, err := AmountFromEUR("1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "400000000" {
		t.Fatalf("satoshis = %s", got)
	}
}
