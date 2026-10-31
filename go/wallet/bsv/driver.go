package bsv

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	bip32 "github.com/bsv-blockchain/go-sdk/compat/bip32"
	"github.com/bsv-blockchain/go-sdk/compat/bip39"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	chaincfg "github.com/bsv-blockchain/go-sdk/transaction/chaincfg"
)

// DriverXPub is the account public key at m/48'/236'/0'.
// The payment address stays on m/44'. The server derives ride keys from this one.
func DriverXPub(mnemonic string) (string, error) {
	account, err := driverAccount(mnemonic)
	if err != nil {
		return "", err
	}
	shared, err := account.Neuter()
	if err != nil {
		return "", fmt.Errorf("bsv: account key: %w", err)
	}
	return shared.String(), nil
}

// SignMessage signs the SHA-256 of msg with the account key at m/48'/236'/0'.
// The result is hex DER. Ride spends use a key derived below this one.
func SignMessage(mnemonic string, msg []byte) (string, error) {
	account, err := driverAccount(mnemonic)
	if err != nil {
		return "", err
	}
	priv, err := account.ECPrivKey()
	if err != nil {
		return "", fmt.Errorf("bsv: account key: %w", err)
	}
	sum := sha256.Sum256(msg)
	sig, err := priv.Sign(sum[:])
	if err != nil {
		return "", fmt.Errorf("bsv: sign: %w", err)
	}
	return hex.EncodeToString(sig.Serialize()), nil
}

// driverPaymentKey is m/48'/236'/0'/0/index, the key that signs a ride payout.
func driverPaymentKey(mnemonic string, index uint32) (*ec.PrivateKey, error) {
	account, err := driverAccount(mnemonic)
	if err != nil {
		return nil, err
	}
	role, err := account.Child(0)
	if err != nil {
		return nil, fmt.Errorf("bsv: %w", err)
	}
	child, err := role.Child(index)
	if err != nil {
		return nil, fmt.Errorf("bsv: %w", err)
	}
	priv, err := child.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("bsv: %w", err)
	}
	return priv, nil
}

func driverAccount(mnemonic string) (*bip32.ExtendedKey, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("bsv: invalid mnemonic")
	}
	master, err := bip32.GenerateHDKeyFromMnemonic(mnemonic, "", &chaincfg.MainNet)
	if err != nil {
		return nil, fmt.Errorf("bsv: %w", err)
	}
	key := master
	for _, index := range []uint32{48, 236, 0} {
		next, err := key.Child(index + bip32.HardenedKeyStart)
		if err != nil {
			return nil, fmt.Errorf("bsv: %w", err)
		}
		key = next
	}
	return key, nil
}
