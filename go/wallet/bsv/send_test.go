package bsv

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		text string
		unit string
		want string
	}{
		{text: "1.5", unit: Whole, want: "150000000"},
		{text: "1,5", unit: Whole, want: "150000000"},
		{text: "1", unit: Whole, want: "100000000"},
		{text: "0.00000001", unit: Whole, want: "1"},
		{text: "1.", unit: Whole, want: "100000000"},
		{text: ".5", unit: Whole, want: "50000000"},
		{text: " 2 ", unit: Atom, want: "2"},
		{text: "00010", unit: Atom, want: "10"},
	}
	for _, tc := range cases {
		got, err := parseAmount(tc.text, tc.unit)
		if err != nil || got != tc.want {
			t.Fatalf("parseAmount(%q, %q) = %q %v", tc.text, tc.unit, got, err)
		}
	}
	for _, text := range []string{"1.000000001", "1.2.3", "-1", "nope", ""} {
		if _, err := parseAmount(text, Whole); err == nil {
			t.Fatalf("parseAmount(%q, bsv) succeeded", text)
		}
	}
	if _, err := parseAmount("1.5", Atom); err == nil {
		t.Fatal("expected satoshi decimal to fail")
	}
	if _, err := parseAmount("1", "ada"); err == nil {
		t.Fatal("expected ada unit to fail")
	}
}

func TestAssembleDropsZeroChange(t *testing.T) {
	account, err := Restore(abandonMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	dest, _, _ := derivePayment(t, abandonMnemonic, 1)
	from, fromScript, err := mainnetAddress(account.Address)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ec.PrivateKeyFromWif(account.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	unlocker, err := p2pkh.Unlock(priv, nil)
	if err != nil {
		t.Fatal(err)
	}
	const total, amount, fee = uint64(100_000), uint64(90_000), uint64(10_000)
	inputs := []utxo{{
		hash:   strings.Repeat("33", 32),
		script: fromScript.String(),
		value:  total,
	}}
	tx, paid, err := assemble(from, dest, inputs, total, amount, fee, unlocker)
	if err != nil {
		t.Fatal(err)
	}
	if paid != fee || len(tx.Outputs) != 1 || tx.Outputs[0].Satoshis != amount {
		t.Fatalf("paid %d outputs %d", paid, len(tx.Outputs))
	}
}

func TestSendRejectsOtherNetworks(t *testing.T) {
	account, err := Restore(abandonMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ec.PrivateKeyFromWif(account.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	testnet, err := script.NewAddressFromPublicKey(priv.PubKey(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, to := range []string{testnet.AddressString, "addr1qy8ac7qqy0vtulyl7wntmsxc6wex80gvcyjy33qffrhm7sh927ysx5sftuw0dlft05dz3c7revpf7jx0xnlcjz3g69mq4afdhv"} {
		err := Send(abandonMnemonic, to, "1")
		if err == nil || !strings.Contains(err.Error(), "address") {
			t.Fatalf("Send(%s) = %v", to, err)
		}
	}
}

func TestSendPaysAndSubmits(t *testing.T) {
	clearBalanceCache()
	clearHistoryCache()
	t.Cleanup(clearBalanceCache)
	t.Cleanup(clearHistoryCache)

	dest, _, _ := derivePayment(t, abandonMnemonic, 1)
	prev := transaction.NewTransaction()
	if err := prev.AddInputFrom(strings.Repeat("11", 32), 0, mustLock(t, dest), 100_000_000, nil); err != nil {
		t.Fatal(err)
	}
	utxoHash := prev.TxID().String()
	const utxoValue = uint64(100_000_000)
	const pay = uint64(25_000_000)

	storeCachedBalance(abandonAddress, "1", "0.00000001 BSV")
	historyMu.Lock()
	historyCache[abandonAddress] = &historyState{txs: map[string]chainTx{}, checked: time.Now().Add(time.Hour)}
	historyMu.Unlock()

	var submits int
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/unspent"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"height": 0, "tx_pos": 0, "tx_hash": utxoHash, "value": 1},
				{"height": 20, "tx_pos": 0, "tx_hash": utxoHash, "value": utxoValue},
			})
		case r.URL.Path == "/tx/raw":
			submits++
			body, _ := io.ReadAll(r.Body)
			var payload struct {
				TxHex string `json:"txhex"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Errorf("broadcast: %v", err)
			}
			raw = payload.TxHex
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"txid":"ok"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	useChain(t, srv)

	if err := Send(abandonMnemonic, dest, "25000000"); err != nil {
		t.Fatal(err)
	}
	if submits != 1 || raw == "" {
		t.Fatalf("submits = %d", submits)
	}
	tx, err := transaction.NewTransactionFromHex(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Inputs) != 1 || tx.Inputs[0].SourceTXID.String() != utxoHash || tx.Inputs[0].SourceTxOutIndex != 0 {
		t.Fatalf("input = %+v", tx.Inputs[0])
	}
	chunks, err := tx.Inputs[0].UnlockingScript.Chunks()
	if err != nil || len(chunks) == 0 || len(chunks[0].Data) == 0 || chunks[0].Data[len(chunks[0].Data)-1] != 0x41 {
		t.Fatalf("sighash = %v %v", chunks, err)
	}
	var paidToDest, paidBack, outputs uint64
	for _, out := range tx.Outputs {
		outputs += out.Satoshis
		if out.LockingScript != nil && out.LockingScript.EqualsHex(mustLock(t, dest)) {
			paidToDest += out.Satoshis
		}
		if out.LockingScript != nil && out.LockingScript.EqualsHex(mustLock(t, abandonAddress)) {
			paidBack += out.Satoshis
		}
	}
	if paidToDest != pay {
		t.Fatalf("paid = %d", paidToDest)
	}
	fee := utxoValue - outputs
	if paidBack != utxoValue-pay-fee {
		t.Fatalf("change = %d fee = %d", paidBack, fee)
	}
	if fee < uint64(tx.Size()) {
		t.Fatalf("fee %d < size %d", fee, tx.Size())
	}
	if _, ok := loadCachedBalance(abandonAddress); ok {
		t.Fatal("balance still cached")
	}
	historyMu.Lock()
	checked := historyCache[abandonAddress].checked
	historyMu.Unlock()
	if !checked.IsZero() {
		t.Fatal("history still cached")
	}
}

func TestSendShortfallDoesNotBroadcast(t *testing.T) {
	var submits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tx/raw" {
			submits++
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"height": 4, "tx_pos": 1, "tx_hash": strings.Repeat("22", 32), "value": 1000},
		})
	}))
	defer srv.Close()
	useChain(t, srv)
	err := Send(abandonMnemonic, abandonAddress, "50000")
	if err == nil || !strings.Contains(err.Error(), "not enough spendable") {
		t.Fatal(err)
	}
	if submits != 0 {
		t.Fatalf("submits = %d", submits)
	}
}
