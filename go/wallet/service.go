package wallet

import (
	"crypto/subtle"
	"errors"
	"strings"
	"sync"

	"github.com/skip2/go-qrcode"
)

var (
	storeMu      sync.RWMutex
	currentStore *Store

	sessionMu sync.Mutex
	session   struct {
		pin string
		key []byte
	}
)

// Init records the Sailjail directories for this process and creates them.
func Init(configDir, dataDir string) error {
	if configDir == "" || dataDir == "" {
		return errors.New("wallet directories are required")
	}
	s := &Store{ConfigDir: configDir, DataDir: dataDir}
	if err := s.ensureDirs(); err != nil {
		return err
	}
	storeMu.Lock()
	currentStore = s
	storeMu.Unlock()
	return nil
}

func openStore() (*Store, error) {
	storeMu.RLock()
	defer storeMu.RUnlock()
	if currentStore == nil {
		return nil, errors.New("wallet storage is not initialized")
	}
	return currentStore, nil
}

func clearSession() {
	sessionMu.Lock()
	session.pin = ""
	session.key = nil
	sessionMu.Unlock()
}

func setSession(pin string, key []byte) {
	sessionMu.Lock()
	session.pin = pin
	session.key = append([]byte(nil), key...)
	sessionMu.Unlock()
}

func sessionKey() ([]byte, error) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if len(session.key) == 0 {
		return nil, errors.New("locked")
	}
	return append([]byte(nil), session.key...), nil
}

func sessionPinMatches(pin string) bool {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if session.pin == "" || len(session.pin) != len(pin) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(session.pin), []byte(pin)) == 1
}

func validPIN(pin string) error {
	if pin == "" {
		return errors.New("enter a pin")
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return errors.New("pin must be digits")
		}
	}
	return nil
}

// PinIsSet reports whether this process's config dir has a PIN verifier.
func PinIsSet() (bool, error) {
	s, err := openStore()
	if err != nil {
		return false, err
	}
	return s.PinIsSet()
}

// SetupPin creates the PIN verifier and keeps the PIN for this process.
func SetupPin(pin, again string) error {
	if err := validPIN(pin); err != nil {
		return err
	}
	if pin != again {
		return errors.New("pin entries do not match")
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	key, err := s.SetupPIN(pin)
	if err != nil {
		return err
	}
	setSession(pin, key)
	return nil
}

// Unlock checks the PIN and keeps it for this process.
func Unlock(pin string) error {
	if err := validPIN(pin); err != nil {
		return err
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	key, err := s.UnlockPIN(pin)
	if err != nil {
		return err
	}
	setSession(pin, key)
	return nil
}

// WalletCount is the number of registered coins.
func WalletCount() int { return len(registry) }

// WalletName is the display name for a wallet index.
func WalletName(index int) (string, error) {
	c, err := coinAt(index)
	if err != nil {
		return "", err
	}
	return c.Name(), nil
}

// WalletIcon is the PNG mark for a wallet index.
func WalletIcon(index int) ([]byte, error) {
	c, err := coinAt(index)
	if err != nil {
		return nil, err
	}
	return c.IconPNG(), nil
}

// WalletExists re-reads that coin's seed file under its lock.
func WalletExists(index int) (bool, error) {
	c, err := coinAt(index)
	if err != nil {
		return false, err
	}
	s, err := openStore()
	if err != nil {
		return false, err
	}
	return s.Exists(c.ID())
}

func walletPublic(index int) (Public, bool, error) {
	c, err := coinAt(index)
	if err != nil {
		return Public{}, false, err
	}
	key, err := sessionKey()
	if err != nil {
		return Public{}, false, err
	}
	s, err := openStore()
	if err != nil {
		return Public{}, false, err
	}
	pub, exists, err := s.Public(c.ID(), key, c.Restore)
	if err != nil || !exists {
		return pub, exists, err
	}
	pub.Details = omitStakeAddress(pub.Details)
	return pub, true, nil
}

// omitStakeAddress drops a stake row stored by earlier builds.
func omitStakeAddress(details []Detail) []Detail {
	kept := make([]Detail, 0, len(details))
	for _, detail := range details {
		if detail.Label == "Stake address" {
			continue
		}
		kept = append(kept, detail)
	}
	return kept
}

// WalletAddress re-reads the receive address under the coin lock.
func WalletAddress(index int) (string, error) {
	pub, exists, err := walletPublic(index)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.New("wallet does not exist")
	}
	return pub.Address, nil
}

// WalletDetailCount re-reads the public rows under the coin lock.
func WalletDetailCount(index int) (int, error) {
	pub, exists, err := walletPublic(index)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errors.New("wallet does not exist")
	}
	return len(pub.Details), nil
}

func walletDetail(index, detail int) (Detail, error) {
	pub, exists, err := walletPublic(index)
	if err != nil {
		return Detail{}, err
	}
	if !exists {
		return Detail{}, errors.New("wallet does not exist")
	}
	if detail < 0 || detail >= len(pub.Details) {
		return Detail{}, errors.New("unknown detail")
	}
	return pub.Details[detail], nil
}

// WalletDetailLabel re-reads one public row label.
func WalletDetailLabel(index, detail int) (string, error) {
	row, err := walletDetail(index, detail)
	if err != nil {
		return "", err
	}
	return row.Label, nil
}

// WalletDetailValue re-reads one public row value.
func WalletDetailValue(index, detail int) (string, error) {
	row, err := walletDetail(index, detail)
	if err != nil {
		return "", err
	}
	return row.Value, nil
}

// WalletQR re-reads the receive address and encodes it as a PNG.
func WalletQR(index int) ([]byte, error) {
	address, err := WalletAddress(index)
	if err != nil {
		return nil, err
	}
	return qrcode.Encode(address, qrcode.Medium, 256)
}

// WalletCreate makes one new wallet for the index and stores it.
func WalletCreate(index int) error {
	c, err := coinAt(index)
	if err != nil {
		return err
	}
	key, err := sessionKey()
	if err != nil {
		return err
	}
	pub, mnemonic, err := c.Create()
	if err != nil {
		return err
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	return s.SaveWallet(c.ID(), key, mnemonic, pub)
}

// WalletRestore stores an existing recovery phrase for the index.
func WalletRestore(index int, mnemonic string) error {
	c, err := coinAt(index)
	if err != nil {
		return err
	}
	key, err := sessionKey()
	if err != nil {
		return err
	}
	mnemonic = strings.TrimSpace(mnemonic)
	pub, err := c.Restore(mnemonic)
	if err != nil {
		return err
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	return s.SaveWallet(c.ID(), key, mnemonic, pub)
}

// WalletMnemonic re-reads and decrypts the recovery phrase.
func WalletMnemonic(index int) (string, error) {
	c, err := coinAt(index)
	if err != nil {
		return "", err
	}
	key, err := sessionKey()
	if err != nil {
		return "", err
	}
	s, err := openStore()
	if err != nil {
		return "", err
	}
	return s.Mnemonic(c.ID(), key)
}

// WalletPrivateKey re-reads the seed and derives the payment extended key.
func WalletPrivateKey(index int) (string, error) {
	c, err := coinAt(index)
	if err != nil {
		return "", err
	}
	mnemonic, err := WalletMnemonic(index)
	if err != nil {
		return "", err
	}
	return c.PrivateKey(mnemonic)
}

// WalletMultisigMaterial re-reads the seed and returns that coin's multisig material.
func WalletMultisigMaterial(index int) (string, error) {
	c, err := coinAt(index)
	if err != nil {
		return "", err
	}
	mnemonic, err := WalletMnemonic(index)
	if err != nil {
		return "", err
	}
	return c.MultisigMaterial(mnemonic)
}

// WalletBalance re-reads the receive address, then asks the coin for its balance.
// The chain query is not done under the coin lock.
func WalletBalance(index int) (string, error) {
	c, err := coinAt(index)
	if err != nil {
		return "", err
	}
	address, err := WalletAddress(index)
	if err != nil {
		return "", err
	}
	return c.Balance(address)
}

// WalletDelete checks the PIN against the one held in this process, then removes the coin's files.
func WalletDelete(index int, pin string) error {
	if !sessionPinMatches(pin) {
		return errors.New("wrong pin")
	}
	c, err := coinAt(index)
	if err != nil {
		return err
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	return s.Delete(c.ID())
}

// WalletSend calls the coin. Cardano returns an error.
func WalletSend(index int, to, amount string) error {
	c, err := coinAt(index)
	if err != nil {
		return err
	}
	if _, err := sessionKey(); err != nil {
		return err
	}
	return c.Send(to, amount)
}
