package wallet

import (
	"path/filepath"
	"strings"
	"testing"
)

func testKDF() KDFParams {
	return KDFParams{Time: 1, Memory: 8 * 1024, Threads: 1}
}

func setupTest(t *testing.T) {
	t.Helper()
	SetKDFParams(testKDF())
	dir := t.TempDir()
	clearSession()
	if err := Init(filepath.Join(dir, "config"), filepath.Join(dir, "data")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		clearSession()
		storeMu.Lock()
		currentStore = nil
		storeMu.Unlock()
		SetKDFParams(ProductionKDF)
	})
}

func TestRegistry(t *testing.T) {
	if WalletCount() != 1 {
		t.Fatalf("count = %d", WalletCount())
	}
	name, err := WalletName(0)
	if err != nil || name != "Cardano" {
		t.Fatalf("name = %q %v", name, err)
	}
	icon, err := WalletIcon(0)
	if err != nil || len(icon) < 8 || string(icon[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal(err)
	}
	if _, err := WalletName(1); err == nil {
		t.Fatal("expected unknown wallet")
	}
}

func TestSetupUnlockCreate(t *testing.T) {
	setupTest(t)
	set, err := PinIsSet()
	if err != nil || set {
		t.Fatalf("pin set = %v %v", set, err)
	}
	if err := SetupPin("12", "13"); err == nil {
		t.Fatal("expected pin mismatch")
	}
	if err := SetupPin("4321", "4321"); err != nil {
		t.Fatal(err)
	}
	set, err = PinIsSet()
	if err != nil || !set {
		t.Fatalf("pin set = %v %v", set, err)
	}
	if err := SetupPin("4321", "4321"); err == nil {
		t.Fatal("expected pin already set")
	}

	clearSession()
	if err := Unlock("0000"); err == nil {
		t.Fatal("expected wrong pin")
	}
	if err := Unlock("4321"); err != nil {
		t.Fatal(err)
	}
	if err := WalletCreate(0); err != nil {
		t.Fatal(err)
	}
	if err := WalletCreate(0); err == nil {
		t.Fatal("expected existing wallet")
	}
	exists, err := WalletExists(0)
	if err != nil || !exists {
		t.Fatalf("exists = %v %v", exists, err)
	}
	address, err := WalletAddress(0)
	if err != nil || !strings.HasPrefix(address, "addr1") {
		t.Fatalf("address = %s %v", address, err)
	}
	n, err := WalletDetailCount(0)
	if err != nil || n != 1 {
		t.Fatalf("details = %d %v", n, err)
	}
	label, err := WalletDetailLabel(0, 0)
	if err != nil || label != "Stake address" {
		t.Fatalf("label = %q %v", label, err)
	}
	stake, err := WalletDetailValue(0, 0)
	if err != nil || !strings.HasPrefix(stake, "stake1") {
		t.Fatalf("stake = %s %v", stake, err)
	}
	mnemonic, err := WalletMnemonic(0)
	if err != nil || len(strings.Fields(mnemonic)) != 24 {
		t.Fatalf("mnemonic words %v", err)
	}
	key, err := WalletPrivateKey(0)
	if err != nil || !strings.HasPrefix(key, "addr_xsk") {
		t.Fatalf("private key %v", err)
	}
	qr, err := WalletQR(0)
	if err != nil || string(qr[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal(err)
	}
	if err := WalletSend(0, address, "1"); err == nil {
		t.Fatal("expected send error")
	}
	if err := WalletDelete(0, "0000"); err == nil {
		t.Fatal("expected wrong pin")
	}
	if err := WalletDelete(0, "4321"); err != nil {
		t.Fatal(err)
	}
	exists, err = WalletExists(0)
	if err != nil || exists {
		t.Fatalf("exists after delete = %v %v", exists, err)
	}
}

func TestRestoreKnownWallet(t *testing.T) {
	setupTest(t)
	if err := SetupPin("7", "7"); err != nil {
		t.Fatal(err)
	}
	const phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if err := WalletRestore(0, "  "+phrase+"\n"); err != nil {
		t.Fatal(err)
	}
	address, err := WalletAddress(0)
	if err != nil {
		t.Fatal(err)
	}
	const want = "addr1qy8ac7qqy0vtulyl7wntmsxc6wex80gvcyjy33qffrhm7sh927ysx5sftuw0dlft05dz3c7revpf7jx0xnlcjz3g69mq4afdhv"
	if address != want {
		t.Fatalf("address = %s", address)
	}
	got, err := WalletMnemonic(0)
	if err != nil || got != phrase {
		t.Fatalf("mnemonic = %q %v", got, err)
	}

	// A new store value, same directories, sees the wallet on disk.
	storeMu.RLock()
	again := &Store{ConfigDir: currentStore.ConfigDir, DataDir: currentStore.DataDir}
	storeMu.RUnlock()
	exists, err := again.Exists("cardano")
	if err != nil || !exists {
		t.Fatalf("reread exists = %v %v", exists, err)
	}
}

func TestLockedWithoutSession(t *testing.T) {
	setupTest(t)
	if err := SetupPin("11", "11"); err != nil {
		t.Fatal(err)
	}
	clearSession()
	if _, err := WalletAddress(0); err == nil {
		t.Fatal("expected locked")
	}
	if err := WalletCreate(0); err == nil {
		t.Fatal("expected locked")
	}
}
