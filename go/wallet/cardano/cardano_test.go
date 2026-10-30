package cardano

import (
	"strings"
	"testing"
)

// Public CIP-1852 vector. Do not use it for funds.
const abandonMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

const (
	abandonMainnetAddress = "addr1qy8ac7qqy0vtulyl7wntmsxc6wex80gvcyjy33qffrhm7sh927ysx5sftuw0dlft05dz3c7revpf7jx0xnlcjz3g69mq4afdhv"
	abandonStakeAddress   = "stake1u8j40zgr2gy4788kl54h6x3gu0pukq5lfr8nflufpg5dzaskqlx2l"
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
	if restored.Address != created.Address || restored.StakeAddress != created.StakeAddress || restored.PrivateKey != created.PrivateKey {
		t.Fatalf("restore mismatch")
	}
}

func TestRestoreRejectsBadMnemonic(t *testing.T) {
	if _, err := Restore("not a real mnemonic phrase"); err == nil {
		t.Fatal("expected invalid mnemonic error")
	}
}

func TestSendReturnsError(t *testing.T) {
	if err := Send("addr1qqqq", "1"); err == nil {
		t.Fatal("expected send error")
	}
}

func TestIconIsPNG(t *testing.T) {
	pngBytes := IconPNG()
	if len(pngBytes) < 8 || string(pngBytes[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("icon is not a png")
	}
}
