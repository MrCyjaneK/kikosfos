package cardano

import (
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blinklabs-io/bursa"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		text string
		unit string
		want string
	}{
		{text: "1.5", unit: "ada", want: "1500000"},
		{text: "1,5", unit: "ada", want: "1500000"},
		{text: "1", unit: "ada", want: "1000000"},
		{text: "0.000001", unit: "ada", want: "1"},
		{text: "1.", unit: "ada", want: "1000000"},
		{text: ".5", unit: "ada", want: "500000"},
		{text: " 2 ", unit: "lovelace", want: "2"},
		{text: "00010", unit: "lovelace", want: "10"},
	}
	for _, tc := range cases {
		got, err := parseAmount(tc.text, tc.unit)
		if err != nil || got != tc.want {
			t.Fatalf("parseAmount(%q, %q) = %q %v", tc.text, tc.unit, got, err)
		}
	}
	for _, text := range []string{"1.0000001", "1.2.3", "-1", "nope", ""} {
		if _, err := parseAmount(text, "ada"); err == nil {
			t.Fatalf("parseAmount(%q, ada) succeeded", text)
		}
	}
	if _, err := parseAmount("1.5", "lovelace"); err == nil {
		t.Fatal("expected lovelace decimal to fail")
	}
}

func TestQuoteUsesCachedPriceWithoutNetwork(t *testing.T) {
	eurRateMu.Lock()
	eurRate.price = nil
	eurRate.until = time.Time{}
	eurRateMu.Unlock()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "unused", http.StatusInternalServerError)
	}))
	defer srv.Close()
	prevClient := balanceClient
	balanceClient = srv.Client()
	t.Cleanup(func() {
		balanceClient = prevClient
		eurRateMu.Lock()
		eurRate.price = nil
		eurRate.until = time.Time{}
		eurRateMu.Unlock()
	})

	ada, lovelace, eur, err := Quote("2", "ada")
	if err != nil {
		t.Fatal(err)
	}
	if ada != "2 ADA" || lovelace != "2000000" || eur != "" {
		t.Fatalf("quote without price = %s %s %s", ada, lovelace, eur)
	}
	if hits != 0 {
		t.Fatalf("quote made %d requests", hits)
	}

	eurRateMu.Lock()
	eurRate.price = big.NewRat(1, 4)
	eurRate.until = time.Now().Add(time.Minute)
	eurRateMu.Unlock()
	ada, lovelace, eur, err = Quote("2", "ada")
	if err != nil {
		t.Fatal(err)
	}
	if ada != "2 ADA" || lovelace != "2000000" || eur != "€0.50" {
		t.Fatalf("quote with price = %s %s %s", ada, lovelace, eur)
	}
	if hits != 0 {
		t.Fatalf("cached quote made %d requests", hits)
	}
}

func TestPreviewUsesAdaPrice(t *testing.T) {
	eurRateMu.Lock()
	eurRate.price = nil
	eurRate.until = time.Time{}
	eurRateMu.Unlock()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":{"200":0.25}}`))
	}))
	defer srv.Close()
	prevRate := eurRateEndpoint
	prevClient := balanceClient
	eurRateEndpoint = srv.URL
	balanceClient = srv.Client()
	t.Cleanup(func() {
		eurRateEndpoint = prevRate
		balanceClient = prevClient
		eurRateMu.Lock()
		eurRate.price = nil
		eurRate.until = time.Time{}
		eurRateMu.Unlock()
	})

	ada, lovelace, eur, err := Preview("2", "ada")
	if err != nil {
		t.Fatal(err)
	}
	if ada != "2 ADA" || lovelace != "2000000" || eur != "€0.50" {
		t.Fatalf("preview = %s %s %s", ada, lovelace, eur)
	}
}

func TestSendRejectsStakeAddress(t *testing.T) {
	err := Send(abandonMnemonic, abandonStakeAddress, "2000000")
	if err == nil || !strings.Contains(err.Error(), "address") {
		t.Fatal(err)
	}
}

func TestSendPaysAndSubmits(t *testing.T) {
	clearBalanceCache()
	t.Cleanup(clearBalanceCache)
	storeCachedBalance(abandonMainnetAddress, "10000000", "10 ADA")
	historyMu.Lock()
	historyCache[abandonMainnetAddress] = &historyState{
		txs:     map[string]chainTx{},
		checked: time.Now().Add(time.Hour),
	}
	historyMu.Unlock()

	var submits int
	var submitted []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/utxos":
			_, _ = w.Write([]byte(`[
				{"tx_hash":"1111111111111111111111111111111111111111111111111111111111111111","tx_index":0,"value":"10000000","asset_list":[]},
				{"tx_hash":"2222222222222222222222222222222222222222222222222222222222222222","tx_index":1,"value":"9000000","asset_list":[{"policy_id":"aa","asset_name":"","quantity":"1"}]}
			]`))
		case "/params":
			_, _ = w.Write([]byte(`[{"epoch_no":100,"min_fee_a":44,"min_fee_b":155381,"coins_per_utxo_size":4310}]`))
		case "/tip":
			_, _ = w.Write([]byte(`[{"abs_slot":5000}]`))
		case "/submit":
			submits++
			if r.Header.Get("Content-Type") != "application/cbor" {
				t.Errorf("content type = %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			submitted = body
			_, _ = w.Write([]byte(`"aabb"`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	restoreSendEndpoints(t, srv)

	if err := Send(abandonMnemonic, abandonMainnetAddress, "2000000"); err != nil {
		t.Fatal(err)
	}
	if submits != 1 {
		t.Fatalf("submits = %d", submits)
	}
	tx, err := conway.NewConwayTransactionFromCbor(submitted)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Inputs()) != 1 {
		t.Fatalf("inputs = %d", len(tx.Inputs()))
	}
	witnesses := tx.Witnesses().Vkey()
	if len(witnesses) != 1 {
		t.Fatalf("witnesses = %d", len(witnesses))
	}
	addr, err := common.NewAddress(abandonMainnetAddress)
	if err != nil {
		t.Fatal(err)
	}
	if common.Blake2b224Hash(witnesses[0].Vkey) != addr.PaymentKeyHash() {
		t.Fatal("witness key does not match the wallet address")
	}
	var paid bool
	var produced uint64
	for _, output := range tx.Outputs() {
		amount := output.Amount().Uint64()
		produced += amount
		if output.Address().String() == abandonMainnetAddress && amount == 2_000_000 {
			paid = true
		}
	}
	if !paid {
		t.Fatal("missing 2 ADA output")
	}
	fee := tx.Fee().Uint64()
	if produced+fee != 10_000_000 {
		t.Fatalf("produced %d fee %d", produced, fee)
	}
	need, err := bursa.MinFee(len(submitted), bursa.ProtocolParams{TxFeePerByte: 44, TxFeeFixed: 155381})
	if err != nil {
		t.Fatal(err)
	}
	if fee < need {
		t.Fatalf("fee %d below minimum %d", fee, need)
	}
	upper, ok := tx.ValidityIntervalUpperBound()
	if !ok || upper != 5000+ttlSlots {
		t.Fatalf("ttl = %d %v", upper, ok)
	}
	if _, cached := loadCachedBalance(abandonMainnetAddress); cached {
		t.Fatal("balance cache still fresh")
	}
	historyMu.Lock()
	checked := historyCache[abandonMainnetAddress].checked
	historyMu.Unlock()
	if !checked.IsZero() {
		t.Fatal("history cache still fresh")
	}
}

func TestSendRejectsShortfallAndDust(t *testing.T) {
	var submits int
	mode := "short"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/utxos":
			if mode == "short" {
				_, _ = w.Write([]byte(`[
					{"tx_hash":"1111111111111111111111111111111111111111111111111111111111111111","tx_index":0,"value":"500000","asset_list":[]},
					{"tx_hash":"2222222222222222222222222222222222222222222222222222222222222222","tx_index":0,"value":"5000000","asset_list":[{"policy_id":"aa","asset_name":"","quantity":"1"}]}
				]`))
				return
			}
			_, _ = w.Write([]byte(`[{"tx_hash":"1111111111111111111111111111111111111111111111111111111111111111","tx_index":0,"value":"10000000","asset_list":[]}]`))
		case "/params":
			_, _ = w.Write([]byte(`[{"epoch_no":100,"min_fee_a":44,"min_fee_b":155381,"coins_per_utxo_size":4310}]`))
		case "/tip":
			_, _ = w.Write([]byte(`[{"abs_slot":5000}]`))
		case "/submit":
			submits++
			_, _ = w.Write([]byte(`"aabb"`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	restoreSendEndpoints(t, srv)

	err := Send(abandonMnemonic, abandonMainnetAddress, "2000000")
	if err == nil || !strings.Contains(err.Error(), "not enough") {
		t.Fatal(err)
	}
	mode = "dust"
	err = Send(abandonMnemonic, abandonMainnetAddress, "1")
	if err == nil || !strings.Contains(err.Error(), "minimum") {
		t.Fatal(err)
	}
	if submits != 0 {
		t.Fatalf("submits = %d", submits)
	}
}

func restoreSendEndpoints(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prevUtxo := utxoEndpoint
	prevParams := epochParamsEndpoint
	prevTip := tipEndpoint
	prevSubmit := submitEndpoint
	prevClient := balanceClient
	utxoEndpoint = srv.URL + "/utxos"
	epochParamsEndpoint = srv.URL + "/params"
	tipEndpoint = srv.URL + "/tip"
	submitEndpoint = srv.URL + "/submit"
	balanceClient = srv.Client()
	t.Cleanup(func() {
		utxoEndpoint = prevUtxo
		epochParamsEndpoint = prevParams
		tipEndpoint = prevTip
		submitEndpoint = prevSubmit
		balanceClient = prevClient
	})
}
