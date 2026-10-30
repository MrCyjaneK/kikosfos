// Package bsv is wallet index 1: one mainnet P2PKH address.
package bsv

import (
	"fmt"
	"strings"

	bip32 "github.com/bsv-blockchain/go-sdk/compat/bip32"
	"github.com/bsv-blockchain/go-sdk/compat/bip39"
	"github.com/bsv-blockchain/go-sdk/script"
	chaincfg "github.com/bsv-blockchain/go-sdk/transaction/chaincfg"

	_ "embed"
)

const (
	Name   = "Bitcoin SV"
	ID     = "bsv"
	Ticker = "BSV"
	Whole  = "bsv"
	Atom   = "satoshi"
)

// Account is one payment key at m/44'/236'/0'/0/0.
type Account struct {
	Mnemonic    string
	Address     string
	PrivateKey  string
	AccountXPub string
}

func Create() (Account, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return Account{}, fmt.Errorf("bsv: generate mnemonic: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return Account{}, fmt.Errorf("bsv: generate mnemonic: %w", err)
	}
	return Restore(mnemonic)
}

func Restore(mnemonic string) (Account, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return Account{}, fmt.Errorf("bsv: invalid mnemonic")
	}
	master, err := bip32.GenerateHDKeyFromMnemonic(mnemonic, "", &chaincfg.MainNet)
	if err != nil {
		return Account{}, fmt.Errorf("bsv: %w", err)
	}
	account, err := accountKey(master)
	if err != nil {
		return Account{}, err
	}
	shared, err := account.Neuter()
	if err != nil {
		return Account{}, fmt.Errorf("bsv: account key: %w", err)
	}
	external, err := account.Child(0)
	if err != nil {
		return Account{}, fmt.Errorf("bsv: %w", err)
	}
	payment, err := external.Child(0)
	if err != nil {
		return Account{}, fmt.Errorf("bsv: %w", err)
	}
	priv, err := payment.ECPrivKey()
	if err != nil {
		return Account{}, fmt.Errorf("bsv: %w", err)
	}
	address, err := script.NewAddressFromPublicKey(priv.PubKey(), true)
	if err != nil {
		return Account{}, fmt.Errorf("bsv: %w", err)
	}
	return Account{
		Mnemonic:    mnemonic,
		Address:     address.AddressString,
		PrivateKey:  priv.Wif(),
		AccountXPub: shared.String(),
	}, nil
}

// accountKey is m/44'/236'/0'. Coin type 236 is BSV.
func accountKey(master *bip32.ExtendedKey) (*bip32.ExtendedKey, error) {
	key := master
	for _, index := range []uint32{44, 236, 0} {
		next, err := key.Child(index + bip32.HardenedKeyStart)
		if err != nil {
			return nil, fmt.Errorf("bsv: %w", err)
		}
		key = next
	}
	return key, nil
}

func IconPNG() []byte {
	return iconPNG
}

//go:embed icon.png
var iconPNG []byte
