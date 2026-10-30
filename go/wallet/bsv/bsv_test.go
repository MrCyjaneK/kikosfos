package bsv

import (
	"strings"
	"testing"

	bip32 "github.com/bsv-blockchain/go-sdk/compat/bip32"
	"github.com/bsv-blockchain/go-sdk/script"
	chaincfg "github.com/bsv-blockchain/go-sdk/transaction/chaincfg"
)

// Public BIP39 vector. Do not use it for funds.
const abandonMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

const (
	abandonAddress = "1K6LZdwpKT5XkEZo2T2kW197aMXYbYMc4f"
	abandonWIF     = "KxU83MzcLXP1WJtoFJXMDMcN3z5ykAa9xLdFTDY5XpV4e6Zit9BA"
	abandonXPub    = "xpub6CdMDgU2hzWyeZ852LWqp5AfDz3ty2cRfi4jEw9BT8aNYugMQvVykQsKLARZdbqKKp7yTviJdL1N9saYLmJNKD1rwVAwLTmU8r8qKeoyG4R"
)

func TestRestoreMatchesSDKPath(t *testing.T) {
	got, err := Restore("  " + abandonMnemonic + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.Mnemonic != abandonMnemonic {
		t.Fatalf("mnemonic = %q", got.Mnemonic)
	}
	if got.Address != abandonAddress || got.PrivateKey != abandonWIF || got.AccountXPub != abandonXPub {
		t.Fatalf("account = %+v", got)
	}
	address, wif, xpub := derivePayment(t, abandonMnemonic, 0)
	if address != abandonAddress || wif != abandonWIF || xpub != abandonXPub {
		t.Fatalf("sdk path = %s %s %s", address, wif, xpub)
	}
}

func TestCreateRoundTrip(t *testing.T) {
	created, err := Create()
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(created.Mnemonic)); n != 24 {
		t.Fatalf("word count = %d, want 24", n)
	}
	if !strings.HasPrefix(created.Address, "1") {
		t.Fatalf("address = %s", created.Address)
	}
	if !strings.HasPrefix(created.PrivateKey, "K") && !strings.HasPrefix(created.PrivateKey, "L") {
		t.Fatalf("wif = %s", created.PrivateKey)
	}
	if !strings.HasPrefix(created.AccountXPub, "xpub") {
		t.Fatalf("xpub = %s", created.AccountXPub)
	}
	restored, err := Restore(created.Mnemonic)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Address != created.Address || restored.PrivateKey != created.PrivateKey || restored.AccountXPub != created.AccountXPub {
		t.Fatal("restore mismatch")
	}
}

func derivePayment(t *testing.T, mnemonic string, index uint32) (address, wif, xpub string) {
	t.Helper()
	master, err := bip32.GenerateHDKeyFromMnemonic(mnemonic, "", &chaincfg.MainNet)
	if err != nil {
		t.Fatal(err)
	}
	key := master
	for _, child := range []uint32{44, 236, 0} {
		key, err = key.Child(child + bip32.HardenedKeyStart)
		if err != nil {
			t.Fatal(err)
		}
	}
	shared, err := key.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	external, err := key.Child(0)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := external.Child(index)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := payment.ECPrivKey()
	if err != nil {
		t.Fatal(err)
	}
	addr, err := script.NewAddressFromPublicKey(priv.PubKey(), true)
	if err != nil {
		t.Fatal(err)
	}
	return addr.AddressString, priv.Wif(), shared.String()
}
