package cardano

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/bursa"
)

func TestSignMessageVerifies(t *testing.T) {
	msg := []byte("kiko-driver-request\n1\nPOST\n/api/v1/driver\n")
	sigHex, err := SignMessage(abandonMnemonic, msg)
	if err != nil {
		t.Fatal(err)
	}
	again, err := SignMessage("  "+abandonMnemonic+"\n", msg)
	if err != nil {
		t.Fatal(err)
	}
	if sigHex != again || len(sigHex) != ed25519.SignatureSize*2 {
		t.Fatalf("signature = %s", sigHex)
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatal(err)
	}
	root, err := bursa.GetRootKeyFromMnemonic(abandonMnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	account, err := bursa.GetMultiSigAccountKey(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	pub := ed25519.PublicKey(account.Public()[:ed25519.PublicKeySize])
	if !ed25519.Verify(pub, msg, sig) {
		t.Fatal("signature does not verify")
	}
	if ed25519.Verify(pub, []byte("other"), sig) {
		t.Fatal("signature verified a different message")
	}
}
