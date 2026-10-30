package bsv

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func clearBalanceCache() {
	balanceCacheMu.Lock()
	balanceCache = map[string]cachedBalance{}
	balanceCacheMu.Unlock()
}

func TestFormatBSV(t *testing.T) {
	cases := []struct {
		satoshis string
		want     string
	}{
		{"0", "0 BSV"},
		{"1", "0.00000001 BSV"},
		{"250000000", "2.5 BSV"},
		{"100000000", "1 BSV"},
		{"100000000000000001", "1000000000.00000001 BSV"},
	}
	for _, tc := range cases {
		got, err := formatBSV(tc.satoshis)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("formatBSV(%s) = %s, want %s", tc.satoshis, got, tc.want)
		}
	}
	if _, err := formatBSV("nope"); err == nil {
		t.Fatal("expected bad amount")
	}
}

func TestReceived(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"confirmed":250000000,"unconfirmed":99}`))
	}))
	defer srv.Close()
	useChain(t, srv)

	ok, err := Received(abandonAddress, "250000000")
	if err != nil || !ok {
		t.Fatalf("held = %v %v", ok, err)
	}
	ok, err = Received(abandonAddress, "250000001")
	if err != nil || ok {
		t.Fatalf("short = %v %v", ok, err)
	}
	ok, err = Received("missing", "1")
	if err != nil || ok {
		t.Fatalf("absent = %v %v", ok, err)
	}
	if hits != 3 {
		t.Fatalf("requests = %d", hits)
	}
	if _, err := Received("", "1"); err == nil {
		t.Fatal("expected a missing address to fail")
	}
}

func TestBalanceReadsConfirmedSatoshis(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/address/1K6LZdwpKT5XkEZo2T2kW197aMXYbYMc4f/balance" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"confirmed":250000000,"unconfirmed":99}`))
	}))
	defer srv.Close()
	useChain(t, srv)

	got, err := Balance(abandonAddress)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.5 BSV" {
		t.Fatalf("balance = %s", got)
	}
	if _, err := Balance(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("requests = %d", hits)
	}
}

func TestBalanceUnknownAddressIsZero(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	useChain(t, srv)
	got, err := Balance(abandonAddress)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0 BSV" {
		t.Fatalf("balance = %s", got)
	}
}

func TestBalanceRejectsNegative(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"confirmed":-1,"unconfirmed":0}`))
	}))
	defer srv.Close()
	useChain(t, srv)
	if _, err := Balance("1neg"); err == nil {
		t.Fatal("expected negative balance to fail")
	}
}
