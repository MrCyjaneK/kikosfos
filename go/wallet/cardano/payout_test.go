package cardano

import (
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blinklabs-io/bursa"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

func TestPayoutSpendsScriptAndLeavesRoomForTheServer(t *testing.T) {
	scriptRaw, err := cbor.Encode(&common.NativeScriptPubkey{Type: 0, Hash: bytes28()})
	if err != nil {
		t.Fatal(err)
	}
	const purse = 20_000_000
	var utxoHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/utxos":
			utxoHits++
			_, _ = w.Write([]byte(`[{"tx_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","tx_index":1,"value":"20000000","asset_list":[]}]`))
		case "/params":
			_, _ = w.Write([]byte(`[{"epoch_no":100,"min_fee_a":44,"min_fee_b":155381,"coins_per_utxo_size":4310}]`))
		case "/tip":
			_, _ = w.Write([]byte(`[{"abs_slot":5000}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	restoreSendEndpoints(t, srv)

	got, err := Payout(abandonMnemonic, abandonMainnetAddress, hex.EncodeToString(scriptRaw), 0)
	if err != nil {
		t.Fatal(err)
	}
	if utxoHits != 1 {
		t.Fatalf("utxo lookups = %d", utxoHits)
	}
	raw, err := hex.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := conway.NewConwayTransactionFromCbor(raw)
	if err != nil {
		t.Fatal(err)
	}
	scripts := tx.Witnesses().NativeScripts()
	if len(scripts) != 1 || hex.EncodeToString(scripts[0].Cbor()) != hex.EncodeToString(scriptRaw) {
		t.Fatalf("script was not kept")
	}
	witnesses := tx.Witnesses().Vkey()
	if len(witnesses) != 1 {
		t.Fatalf("witnesses = %d", len(witnesses))
	}
	if !ed25519.Verify(ed25519.PublicKey(witnesses[0].Vkey), tx.Hash().Bytes(), witnesses[0].Signature) {
		t.Fatal("driver signature does not verify")
	}
	var paid uint64
	for _, output := range tx.Outputs() {
		if output.Address().String() == abandonMainnetAddress {
			paid += output.Amount().Uint64()
		}
	}
	fee := tx.Fee().Uint64()
	if paid+fee != purse {
		t.Fatalf("paid %d fee %d", paid, fee)
	}
	need, err := bursa.MinFee(len(raw)+serverWitnessBytes, bursa.ProtocolParams{TxFeePerByte: 44, TxFeeFixed: 155381})
	if err != nil {
		t.Fatal(err)
	}
	if fee < need {
		t.Fatalf("fee %d does not cover the server witness (%d)", fee, need)
	}
}

func TestPayoutRejectsABadScript(t *testing.T) {
	if _, err := Payout(abandonMnemonic, abandonMainnetAddress, "zz", 0); err == nil {
		t.Fatal("expected a bad script")
	}
}

func bytes28() []byte {
	out := make([]byte, 28)
	for i := range out {
		out[i] = byte(i + 1)
	}
	return out
}
