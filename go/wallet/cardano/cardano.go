// Package cardano is wallet index 0: one Shelley account on mainnet.
package cardano

import (
	"fmt"
	"strings"

	"github.com/blinklabs-io/bursa"
	"github.com/btcsuite/btcd/btcutil/bech32"

	_ "embed"
)

const (
	Name = "Cardano"
	ID   = "cardano"
)

type Account struct {
	Mnemonic         string
	Address          string
	StakeAddress     string
	PrivateKey       string
	AccountSharedXVK string
}

func Create() (Account, error) {
	mnemonic, err := bursa.GenerateMnemonic()
	if err != nil {
		return Account{}, fmt.Errorf("cardano: generate mnemonic: %w", err)
	}
	return Restore(mnemonic)
}

func Restore(mnemonic string) (Account, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	created, err := bursa.NewWallet(mnemonic, bursa.WithNetwork("mainnet"))
	if err != nil {
		return Account{}, fmt.Errorf("cardano: %w", err)
	}
	shared, err := accountSharedXVK(mnemonic)
	if err != nil {
		return Account{}, err
	}
	return Account{
		Mnemonic:         mnemonic,
		Address:          created.PaymentAddress,
		StakeAddress:     created.StakeAddress,
		PrivateKey:       created.PaymentExtendedSKey.String(),
		AccountSharedXVK: shared,
	}, nil
}

func accountSharedXVK(mnemonic string) (string, error) {
	root, err := bursa.GetRootKeyFromMnemonic(mnemonic, "")
	if err != nil {
		return "", fmt.Errorf("cardano: %w", err)
	}
	account, err := bursa.GetMultiSigAccountKey(root, 0)
	if err != nil {
		return "", fmt.Errorf("cardano: account key: %w", err)
	}
	encoded, err := encodeBech32("acct_shared_xvk", account.Public())
	if err != nil {
		return "", fmt.Errorf("cardano: account key: %w", err)
	}
	return encoded, nil
}

func encodeBech32(hrp string, payload []byte) (string, error) {
	converted, err := bech32.ConvertBits(payload, 8, 5, true)
	if err != nil {
		return "", err
	}
	return bech32.Encode(hrp, converted)
}

func IconPNG() []byte {
	return iconPNG
}

//go:embed icon.png
var iconPNG []byte
