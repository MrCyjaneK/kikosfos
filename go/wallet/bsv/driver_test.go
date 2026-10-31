package bsv

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
)

func TestDriverKeySigns(t *testing.T) {
	msg := []byte("kiko-driver-request\n1\nGET\n/api/v1/driver/drive/available\n")
	sigHex, err := SignMessage(abandonMnemonic, msg)
	if err != nil {
		t.Fatal(err)
	}
	again, err := SignMessage(abandonMnemonic, msg)
	if err != nil || again != sigHex {
		t.Fatalf("signature = %s %s %v", sigHex, again, err)
	}
	xpub, err := DriverXPub(abandonMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(xpub, "xpub") || xpub == abandonXPub {
		t.Fatalf("driver xpub = %s", xpub)
	}
	account, err := driverAccount(abandonMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := account.ECPrivKey()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := ec.ParseDERSignature(raw)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(msg)
	if !sig.Verify(sum[:], priv.PubKey()) {
		t.Fatal("signature does not verify")
	}
}
