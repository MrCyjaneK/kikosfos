package cardano

import (
	"encoding/hex"
	"strings"
	"testing"
)

// Public CIP-1852 vector. Do not use it for funds.
const abandonMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

const (
	abandonMainnetAddress = "addr1qy8ac7qqy0vtulyl7wntmsxc6wex80gvcyjy33qffrhm7sh927ysx5sftuw0dlft05dz3c7revpf7jx0xnlcjz3g69mq4afdhv"
	abandonStakeAddress   = "stake1u8j40zgr2gy4788kl54h6x3gu0pukq5lfr8nflufpg5dzaskqlx2l"
	abandonSharedXVK      = "acct_shared_xvk1amxjt3n7l964nlpgqtzw72uy299k6txq35pdj6egf3j6huhjatdxtdqv95hyyrnttxma0r95wmygj49pz0axhjwj93pwt0xt7f080gqvmygft"
)

func TestRestoreMatchesCardanoVector(t *testing.T) {
	got, err := Restore("  " + abandonMnemonic + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.Mnemonic != abandonMnemonic {
		t.Fatalf("mnemonic = %q", got.Mnemonic)
	}
	if got.Address != abandonMainnetAddress {
		t.Fatalf("address = %s", got.Address)
	}
	if got.StakeAddress != abandonStakeAddress {
		t.Fatalf("stake address = %s", got.StakeAddress)
	}
	if !strings.HasPrefix(got.PrivateKey, "addr_xsk") {
		t.Fatalf("private key = %s", got.PrivateKey)
	}
	if got.AccountSharedXVK != abandonSharedXVK {
		t.Fatalf("account key = %s", got.AccountSharedXVK)
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
	if !strings.HasPrefix(created.Address, "addr1") {
		t.Fatalf("address = %s", created.Address)
	}
	if !strings.HasPrefix(created.StakeAddress, "stake1") {
		t.Fatalf("stake address = %s", created.StakeAddress)
	}
	restored, err := Restore(created.Mnemonic)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Address != created.Address || restored.StakeAddress != created.StakeAddress || restored.PrivateKey != created.PrivateKey || restored.AccountSharedXVK != created.AccountSharedXVK {
		t.Fatalf("restore mismatch")
	}
	if !strings.HasPrefix(created.AccountSharedXVK, "acct_shared_xvk") {
		t.Fatalf("account key = %s", created.AccountSharedXVK)
	}
}

func TestEncodeBech32MatchesCIP1854(t *testing.T) {
	raw, err := hex.DecodeString("179be1ac9e63abb5fe4666df03007b0733a3b63cdd08cd3635b8d364e16aaf2649856f2ba513786afdbe5c7a07565c029ba7a4290d20aa3c80ecc1835841ed78")
	if err != nil {
		t.Fatal(err)
	}
	got, err := encodeBech32("addr_shared_xvk", raw)
	if err != nil {
		t.Fatal(err)
	}
	const want = "addr_shared_xvk1z7d7rty7vw4mtljxvm0sxqrmque68d3um5yv6d34hrfkfct24unynpt09wj3x7r2lkl9c7s82ewq9xa85s5s6g928jqwesvrtpq767qs66fnu"
	if got != want {
		t.Fatalf("encoded = %s", got)
	}
}

func TestRestoreRejectsBadMnemonic(t *testing.T) {
	if _, err := Restore("not a real mnemonic phrase"); err == nil {
		t.Fatal("expected invalid mnemonic error")
	}
}

func TestIconIsPNG(t *testing.T) {
	pngBytes := IconPNG()
	if len(pngBytes) < 8 || string(pngBytes[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("icon is not a png")
	}
}
