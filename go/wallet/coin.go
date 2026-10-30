// Package wallet is the coin-agnostic wallet service behind the app.
//
// Wallet index 0 is Cardano. The next coin takes the next index and stays there.
// The process keeps the PIN and its derived key in memory after unlock.
// Wallet records are read from disk under a file lock on every call.
package wallet

import (
	"fmt"

	"kiko-sfos/wallet/cardano"
)

// Detail is one public row a wallet can show under its receive address.
type Detail struct {
	Label string
	Value string
}

// Public is the non-secret view of a wallet.
type Public struct {
	Address string
	Details []Detail
}

// Coin is one currency. Adding a coin means implementing this and appending it
// to the registry.
type Coin interface {
	ID() string
	Name() string
	IconPNG() []byte
	Create() (Public, string, error)
	Restore(mnemonic string) (Public, error)
	PrivateKey(mnemonic string) (string, error)
	Send(to, amount string) error
}

type cardanoCoin struct{}

func (cardanoCoin) ID() string { return cardano.ID }

func (cardanoCoin) Name() string { return cardano.Name }

func (cardanoCoin) IconPNG() []byte { return cardano.IconPNG() }

func (cardanoCoin) Create() (Public, string, error) {
	account, err := cardano.Create()
	if err != nil {
		return Public{}, "", err
	}
	return publicFrom(account), account.Mnemonic, nil
}

func (cardanoCoin) Restore(mnemonic string) (Public, error) {
	account, err := cardano.Restore(mnemonic)
	if err != nil {
		return Public{}, err
	}
	return publicFrom(account), nil
}

func (cardanoCoin) PrivateKey(mnemonic string) (string, error) {
	account, err := cardano.Restore(mnemonic)
	if err != nil {
		return "", err
	}
	return account.PrivateKey, nil
}

func (cardanoCoin) Send(to, amount string) error {
	return cardano.Send(to, amount)
}

func publicFrom(account cardano.Account) Public {
	return Public{
		Address: account.Address,
		Details: []Detail{{
			Label: "Stake address",
			Value: account.StakeAddress,
		}},
	}
}

// registry order is the stable wallet index. Do not reorder it.
var registry = []Coin{
	cardanoCoin{},
}

func coinAt(index int) (Coin, error) {
	if index < 0 || index >= len(registry) {
		return nil, fmt.Errorf("unknown wallet %d", index)
	}
	return registry[index], nil
}
