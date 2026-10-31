package bsv

import (
	"testing"

	"github.com/bsv-blockchain/go-sdk/script"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
)

func TestPayoutUnlocksWithOneSignature(t *testing.T) {
	account, err := Restore(abandonMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	_, lock, err := mainnetAddress(account.Address)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []utxo{{
		hash:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		script: lock.String(),
		index:  0,
		value:  50_000,
	}}
	tx, err := buildPayout(abandonMnemonic, lock.String(), 4, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Inputs) != 1 || len(tx.Outputs) != 1 {
		t.Fatalf("shape = %d inputs %d outputs", len(tx.Inputs), len(tx.Outputs))
	}
	chunks, err := tx.Inputs[0].UnlockingScript.Chunks()
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 || chunks[0].Op != script.Op0 || len(chunks[1].Data) < 2 {
		t.Fatalf("unlocking script = %+v", chunks)
	}
	sig := chunks[1].Data
	if sig[len(sig)-1] != byte(sighash.AllForkID) {
		t.Fatalf("sighash = %x", sig[len(sig)-1])
	}
	final := uint64(tx.Size()) + secondSigBytes
	got := inputs[0].value - tx.Outputs[0].Satoshis
	if got < final {
		t.Fatalf("fee %d does not cover a second signature (%d)", got, final)
	}
	ef, err := tx.EFHex()
	if err != nil || ef == "" {
		t.Fatal(err)
	}
}
