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
		resetGPS()
		resetDrives()
		clearSession()
		storeMu.Lock()
		currentStore = nil
		storeMu.Unlock()
		SetKDFParams(ProductionKDF)
	})
}

func TestRegistry(t *testing.T) {
	if WalletCount() != 2 {
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
	whole, err := WalletWhole(0)
	if err != nil || whole != "ada" {
		t.Fatalf("whole = %q %v", whole, err)
	}
	atom, err := WalletAtom(0)
	if err != nil || atom != "lovelace" {
		t.Fatalf("atom = %q %v", atom, err)
	}
	name, err = WalletName(1)
	if err != nil || name != "Bitcoin SV" {
		t.Fatalf("name = %q %v", name, err)
	}
	icon, err = WalletIcon(1)
	if err != nil || len(icon) < 8 || string(icon[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal(err)
	}
	if _, err := WalletName(2); err == nil {
		t.Fatal("expected unknown wallet")
	}
}

func TestBSVQuoteUnits(t *testing.T) {
	text, sats, eur, err := WalletQuote(1, "1.5", "bsv")
	if err != nil {
		t.Fatal(err)
	}
	if text != "1.5 BSV" || sats != "150000000" || eur != "" {
		t.Fatalf("quote = %s %s %s", text, sats, eur)
	}
	_, sats, _, err = WalletQuote(1, "2", "satoshi")
	if err != nil || sats != "2" {
		t.Fatalf("satoshi quote = %s %v", sats, err)
	}
	if _, _, _, err := WalletQuote(1, "1", "ada"); err == nil {
		t.Fatal("expected ada unit to fail")
	}
	text, sats, _, err = WalletQuote(0, "1.5", "ada")
	if err != nil || text != "1.5 ADA" || sats != "1500000" {
		t.Fatalf("cardano quote = %s %s %v", text, sats, err)
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
	if err != nil || n != 0 {
		t.Fatalf("details = %d %v", n, err)
	}
	mnemonic, err := WalletMnemonic(0)
	if err != nil || len(strings.Fields(mnemonic)) != 24 {
		t.Fatalf("mnemonic words %v", err)
	}
	key, err := WalletPrivateKey(0)
	if err != nil || !strings.HasPrefix(key, "addr_xsk") {
		t.Fatalf("private key %v", err)
	}
	material, err := WalletMultisigMaterial(0)
	if err != nil || !strings.HasPrefix(material, "acct_shared_xvk") {
		t.Fatalf("multisig material %s %v", material, err)
	}
	qr, err := WalletQR(0)
	if err != nil || string(qr[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal(err)
	}
	if err := WalletSend(0, address, "0"); err == nil {
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

func TestOmitStoredStakeAddress(t *testing.T) {
	got := omitStakeAddress([]Detail{
		{Label: "Stake address", Value: "stake1u"},
		{Label: "Account", Value: "0"},
	})
	if len(got) != 1 || got[0].Label != "Account" {
		t.Fatalf("details = %+v", got)
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
