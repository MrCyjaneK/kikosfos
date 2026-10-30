package wallet

import (
	"strings"
	"testing"
)

// Public CIP-1852 vector. Do not use it for funds.
// Mnemonic derives payment m/1852'/1815'/0'/0/0 and stake m/1852'/1815'/0'/2/0.
const abandonMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

const (
	abandonMainnetAddress = "addr1qy8ac7qqy0vtulyl7wntmsxc6wex80gvcyjy33qffrhm7sh927ysx5sftuw0dlft05dz3c7revpf7jx0xnlcjz3g69mq4afdhv"
	abandonStakeAddress   = "stake1u8j40zgr2gy4788kl54h6x3gu0pukq5lfr8nflufpg5dzaskqlx2l"
)

func TestRestoreMatchesCardanoVector(t *testing.T) {
	got, err := Restore("  "+abandonMnemonic+"\n", Mainnet)
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
}

func TestGenerateRoundTrip(t *testing.T) {
	created, err := Generate(Preprod)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(created.Mnemonic)); n != 24 {
		t.Fatalf("word count = %d, want 24", n)
	}
	if !strings.HasPrefix(created.Address, "addr_test1") {
		t.Fatalf("address = %s", created.Address)
	}
	if !strings.HasPrefix(created.StakeAddress, "stake_test1") {
		t.Fatalf("stake address = %s", created.StakeAddress)
	}

	restored, err := Restore(created.Mnemonic, Preprod)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Address != created.Address || restored.StakeAddress != created.StakeAddress {
		t.Fatalf("restore mismatch:\n created %#v\nrestored %#v", created, restored)
	}

	mainnet, err := Restore(created.Mnemonic, Mainnet)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.Address == created.Address || !strings.HasPrefix(mainnet.Address, "addr1") {
		t.Fatalf("mainnet address = %s", mainnet.Address)
	}
}

func TestRestoreRejectsBadInput(t *testing.T) {
	if _, err := Restore("not a real mnemonic phrase", Mainnet); err == nil {
		t.Fatal("expected invalid mnemonic error")
	}
	if _, err := Restore(abandonMnemonic, Network("nope")); err == nil {
		t.Fatal("expected invalid network error")
	}
}
