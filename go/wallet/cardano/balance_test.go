package cardano

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFormatADA(t *testing.T) {
	cases := []struct {
		lovelace string
		want     string
	}{
		{"0", "0 ADA"},
		{"1", "0.000001 ADA"},
		{"1500000", "1.5 ADA"},
		{"1000000", "1 ADA"},
		{"1000000000001", "1000000.000001 ADA"},
	}
	for _, tc := range cases {
		got, err := formatADA(tc.lovelace)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("formatADA(%s) = %s, want %s", tc.lovelace, got, tc.want)
		}
	}
	if _, err := formatADA("nope"); err == nil {
		t.Fatal("expected bad amount")
	}
}

func clearBalanceCache() {
	balanceCacheMu.Lock()
	balanceCache = map[string]cachedBalance{}
	balanceCacheMu.Unlock()
}

func TestReceived(t *testing.T) {
	mainHits, preHits := 0, 0
	mainSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mainHits++
		_, _ = w.Write([]byte(`[{"balance":"100"}]`))
	}))
	defer mainSrv.Close()
	preSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		preHits++
		_, _ = w.Write([]byte(`[{"balance":"5000000"}]`))
	}))
	defer preSrv.Close()
	prevMain, prevPre, prevClient := balanceEndpoint, preprodBalanceEndpoint, balanceClient
	balanceEndpoint = mainSrv.URL
	preprodBalanceEndpoint = preSrv.URL
	balanceClient = &http.Client{}
	t.Cleanup(func() {
		balanceEndpoint = prevMain
		preprodBalanceEndpoint = prevPre
		balanceClient = prevClient
	})

	ok, err := Received("addr1qx", "100")
	if err != nil || !ok || mainHits != 1 || preHits != 0 {
		t.Fatalf("mainnet = %v %v hits %d %d", ok, err, mainHits, preHits)
	}
	ok, err = Received("addr1qx", "101")
	if err != nil || ok {
		t.Fatalf("short = %v %v", ok, err)
	}
	ok, err = Received("addr_test1qx", "5000000")
	if err != nil || !ok || preHits != 1 {
		t.Fatalf("preprod = %v %v hits %d", ok, err, preHits)
	}
	if _, err := Received("", "1"); err == nil {
		t.Fatal("expected a missing address to fail")
	}
	if _, err := Received("addr1qx", "-1"); err == nil {
		t.Fatal("expected a negative amount to fail")
	}
}

func TestBalanceReadsIndexer(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body struct {
			Addresses []string `json:"_addresses"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Addresses) != 1 || body.Addresses[0] != "addr1test" {
			t.Errorf("addresses = %v", body.Addresses)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"address":"addr1test","balance":"2500000"}]`))
	}))
	defer srv.Close()

	prevEndpoint := balanceEndpoint
	prevClient := balanceClient
	balanceEndpoint = srv.URL
	balanceClient = srv.Client()
	t.Cleanup(func() {
		balanceEndpoint = prevEndpoint
		balanceClient = prevClient
	})

	got, err := Balance("addr1test")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.5 ADA" {
		t.Fatalf("balance = %s", got)
	}
}

func TestBalanceEmptyAddressIsZero(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	prevEndpoint := balanceEndpoint
	prevClient := balanceClient
	balanceEndpoint = srv.URL
	balanceClient = srv.Client()
	t.Cleanup(func() {
		balanceEndpoint = prevEndpoint
		balanceClient = prevClient
	})

	got, err := Balance("addr1empty")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0 ADA" {
		t.Fatalf("balance = %s", got)
	}
}

func TestBalanceCacheSkipsRepeatRequest(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`[{"address":"addr1cache","balance":"1000000"}]`))
	}))
	defer srv.Close()

	prevEndpoint := balanceEndpoint
	prevClient := balanceClient
	balanceEndpoint = srv.URL
	balanceClient = srv.Client()
	t.Cleanup(func() {
		balanceEndpoint = prevEndpoint
		balanceClient = prevClient
	})

	got, err := Balance("addr1cache")
	if err != nil {
		t.Fatal(err)
	}
	again, err := Balance("addr1cache")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1 ADA" || again != got {
		t.Fatalf("balance = %s then %s", got, again)
	}
	if hits != 1 {
		t.Fatalf("requests = %d", hits)
	}

	balanceCacheMu.Lock()
	entry := balanceCache["addr1cache"]
	entry.until = time.Now().Add(-time.Second)
	balanceCache["addr1cache"] = entry
	balanceCacheMu.Unlock()

	if _, err := Balance("addr1cache"); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("requests after expiry = %d", hits)
	}
}
