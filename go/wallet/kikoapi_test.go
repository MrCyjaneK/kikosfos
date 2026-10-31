package wallet

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func serveKiko(t *testing.T, h http.Handler) {
	t.Helper()
	if h == nil {
		h = kikoHandler(nil)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	prev := apiBase
	apiBase = func(string) string { return srv.URL }
	t.Cleanup(func() { apiBase = prev })
}

func serveKikoHosts(t *testing.T, hosts map[string]http.Handler) {
	t.Helper()
	origins := make(map[string]string, len(hosts))
	for host, h := range hosts {
		srv := httptest.NewServer(h)
		t.Cleanup(srv.Close)
		origins[host] = srv.URL
	}
	prev := apiBase
	apiBase = func(host string) string {
		if origin, ok := origins[host]; ok {
			return origin
		}
		return defaultAPIBase(host)
	}
	t.Cleanup(func() { apiBase = prev })
}

func kikoHandler(available http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/currency":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"currency":"cardano"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/driver":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.Copy(w, r.Body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/driver/drive/available" && available != nil:
			available(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/driver/drive/available":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		default:
			http.NotFound(w, r)
		}
	})
}

func TestDefaultAPIBase(t *testing.T) {
	if got := defaultAPIBase("ride.example.com"); got != "https://ride.example.com" {
		t.Fatalf("host = %s", got)
	}
	if got := defaultAPIBase("ride.example.com:8443"); got != "https://ride.example.com:8443" {
		t.Fatalf("port = %s", got)
	}
	if got := defaultAPIBase("127.0.0.1:8080"); got != "http://127.0.0.1:8080" {
		t.Fatalf("loopback = %s", got)
	}
	if got := defaultAPIBase("localhost:8080"); got != "http://localhost:8080" {
		t.Fatalf("localhost = %s", got)
	}
}

func TestNormalizeHostPort(t *testing.T) {
	got, err := normalizeHost(" 127.0.0.1:8080 ")
	if err != nil || got != "127.0.0.1:8080" {
		t.Fatalf("port = %s %v", got, err)
	}
	if _, err := normalizeHost("example.com:nope"); err == nil {
		t.Fatal("expected a bad port to be rejected")
	}
	if _, err := normalizeHost("example.com:0"); err == nil {
		t.Fatal("expected port 0 to be rejected")
	}
}

func TestLocalCurrency(t *testing.T) {
	setupTest(t)
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"currency":"cardano"}`))
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := AddServer(u.Host); err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/currency" {
		t.Fatalf("path = %s", path)
	}
	host, err := ServerHost(0)
	currency, err2 := ServerCurrency(0)
	if err != nil || err2 != nil || host != u.Host || currency != "cardano" {
		t.Fatalf("server = %s %s %v %v", host, currency, err, err2)
	}
}

func TestDiscoverCurrencyFailure(t *testing.T) {
	setupTest(t)
	if err := SetupPin("42", "42"); err != nil {
		t.Fatal(err)
	}
	serveKiko(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"could not geocode"}`))
	}))
	err := AddServer("example.com")
	if err == nil || !strings.Contains(err.Error(), "could not geocode") {
		t.Fatal(err)
	}
	n, err := ServerCount()
	if err != nil || n != 0 {
		t.Fatalf("count = %d %v", n, err)
	}
}

func TestRegisterDriverConflict(t *testing.T) {
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
	posts := 0
	var signedErr string
	var signedMu sync.Mutex
	note := func(r *http.Request, body []byte, wantKey bool) {
		if err := verifySigned(r, body, 0, wantKey); err != nil {
			signedMu.Lock()
			signedErr = err.Error()
			signedMu.Unlock()
		}
	}
	serveKiko(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/currency":
			_, _ = w.Write([]byte(`{"currency":"cardano"}`))
		case "/api/v1/driver":
			posts++
			note(r, body, false)
			if posts > 1 {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = w.Write(body)
		case "/api/v1/driver/drive/available":
			note(r, body, true)
			_, _ = w.Write([]byte("[]"))
		default:
			http.NotFound(w, r)
		}
	}))
	if err := AddServer("ride.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := SaveServerFees(0, "2", "0.5", "ada"); err != nil {
		t.Fatal(err)
	}
	msg, err := RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("register: %q %v", msg, err)
	}
	msg, err = RegisterDrivers()
	if err != nil || msg != "" {
		t.Fatalf("update: %q %v", msg, err)
	}
	if posts != 2 {
		t.Fatalf("posts = %d", posts)
	}
	signedMu.Lock()
	errText := signedErr
	signedMu.Unlock()
	if errText != "" {
		t.Fatal(errText)
	}
	if err := RefreshDrives(5); err != nil {
		t.Fatal(err)
	}
	signedMu.Lock()
	errText = signedErr
	signedMu.Unlock()
	if errText != "" {
		t.Fatal(errText)
	}
	if n := DriveCount(); n != 0 {
		t.Fatalf("count = %d", n)
	}
}

func verifySigned(r *http.Request, body []byte, coin int, wantKey bool) error {
	if wantKey {
		if r.Header.Get("X-Driver-Key") == "" {
			return fmt.Errorf("%s sent no driver key", r.URL.Path)
		}
	} else if r.Header.Get("X-Driver-Key") != "" {
		return fmt.Errorf("%s sent a driver key", r.URL.Path)
	}
	ts, err := strconv.ParseInt(r.Header.Get("X-Driver-Timestamp"), 10, 64)
	if err != nil {
		return fmt.Errorf("%s timestamp: %w", r.URL.Path, err)
	}
	sig := r.Header.Get("X-Driver-Signature")
	if sig == "" {
		return fmt.Errorf("%s sent no signature", r.URL.Path)
	}
	got, err := signDriver(coin, driverMessage(ts, r.Method, r.URL.RequestURI(), body))
	if err != nil {
		return err
	}
	if got != sig {
		return fmt.Errorf("%s signature does not match", r.URL.Path)
	}
	return nil
}
