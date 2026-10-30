// Package wallet is the coin-agnostic wallet service behind the app.
//
// Wallet index 0 is Cardano. The next coin takes the next index and stays there.
// The process keeps the PIN and its derived key in memory after unlock.
// Wallet records are read from disk under a file lock on every call.
package wallet

import (
	"fmt"

	"kiko-sfos/wallet/bsv"
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

// Activity is one confirmed movement of the wallet's funds.
type Activity struct {
	Direction string
	Amount    string
	When      string
}

// Coin is one currency. Adding a coin means implementing this and appending it
// to the registry.
type Coin interface {
	ID() string
	Name() string
	// Ticker is the price symbol, such as ADA.
	// A server's currency matches ID, not Ticker.
	Ticker() string
	// Whole is the unit id typed for a coin amount, such as "ada" or "bsv".
	Whole() string
	// Atom is the smallest unit id, such as "lovelace" or "satoshi".
	Atom() string
	IconPNG() []byte
	Create() (Public, string, error)
	Restore(mnemonic string) (Public, error)
	PrivateKey(mnemonic string) (string, error)
	// MultisigMaterial is the shareable key for a multisig setup.
	// Each coin picks its own encoding. Cardano uses acct_shared_xvk.
	MultisigMaterial(mnemonic string) (string, error)
	// Balance is the on-chain amount for an address, formatted for display.
	Balance(address string) (string, error)
	// Fiat quotes that balance. Empty when the coin has no euro price.
	Fiat(address string) (string, error)
	// RefreshActivity loads transactions that are not already cached.
	RefreshActivity(address string) error
	// Activities is the cached list and does not use the network.
	Activities(address string) []Activity
	// Quote formats an amount typed in Whole or Atom without the network.
	// The euro text is empty until a price is cached.
	Quote(amount, unit string) (ada, lovelace, eur string, err error)
	// WarmEuro fetches the euro price so a later Quote can show it.
	WarmEuro() error
	// AmountFromEUR converts a euro amount into this coin's smallest unit.
	// It uses the network when the price is not cached.
	AmountFromEUR(eur string) (string, error)
	// Send pays amount lovelace to a payment address and broadcasts it.
	Send(mnemonic, to, amount string) error
}

type cardanoCoin struct{}

func (cardanoCoin) ID() string { return cardano.ID }

func (cardanoCoin) Name() string { return cardano.Name }

func (cardanoCoin) Ticker() string { return cardano.Ticker }

func (cardanoCoin) Whole() string { return "ada" }

func (cardanoCoin) Atom() string { return "lovelace" }

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

func (cardanoCoin) MultisigMaterial(mnemonic string) (string, error) {
	account, err := cardano.Restore(mnemonic)
	if err != nil {
		return "", err
	}
	return account.AccountSharedXVK, nil
}

func (cardanoCoin) Balance(address string) (string, error) {
	return cardano.Balance(address)
}

func (cardanoCoin) Fiat(address string) (string, error) {
	return cardano.Fiat(address)
}

func (cardanoCoin) RefreshActivity(address string) error {
	return cardano.RefreshActivity(address)
}

func (cardanoCoin) Activities(address string) []Activity {
	rows := cardano.Activities(address)
	out := make([]Activity, len(rows))
	for i, row := range rows {
		out[i] = Activity{Direction: row.Direction, Amount: row.Amount, When: row.When}
	}
	return out
}

func (cardanoCoin) Quote(amount, unit string) (string, string, string, error) {
	return cardano.Quote(amount, unit)
}

func (cardanoCoin) WarmEuro() error {
	_, err := cardano.WarmEuro()
	return err
}

func (cardanoCoin) AmountFromEUR(eur string) (string, error) {
	return cardano.AmountFromEUR(eur)
}

func (cardanoCoin) Send(mnemonic, to, amount string) error {
	return cardano.Send(mnemonic, to, amount)
}

type bsvCoin struct{}

func (bsvCoin) ID() string { return bsv.ID }

func (bsvCoin) Name() string { return bsv.Name }

func (bsvCoin) Ticker() string { return bsv.Ticker }

func (bsvCoin) Whole() string { return bsv.Whole }

func (bsvCoin) Atom() string { return bsv.Atom }

func (bsvCoin) IconPNG() []byte { return bsv.IconPNG() }

func (bsvCoin) Create() (Public, string, error) {
	account, err := bsv.Create()
	if err != nil {
		return Public{}, "", err
	}
	return Public{Address: account.Address}, account.Mnemonic, nil
}

func (bsvCoin) Restore(mnemonic string) (Public, error) {
	account, err := bsv.Restore(mnemonic)
	if err != nil {
		return Public{}, err
	}
	return Public{Address: account.Address}, nil
}

func (bsvCoin) PrivateKey(mnemonic string) (string, error) {
	account, err := bsv.Restore(mnemonic)
	if err != nil {
		return "", err
	}
	return account.PrivateKey, nil
}

func (bsvCoin) MultisigMaterial(mnemonic string) (string, error) {
	account, err := bsv.Restore(mnemonic)
	if err != nil {
		return "", err
	}
	return account.AccountXPub, nil
}

func (bsvCoin) Balance(address string) (string, error) {
	return bsv.Balance(address)
}

func (bsvCoin) Fiat(address string) (string, error) {
	return bsv.Fiat(address)
}

func (bsvCoin) RefreshActivity(address string) error {
	return bsv.RefreshActivity(address)
}

func (bsvCoin) Activities(address string) []Activity {
	rows := bsv.Activities(address)
	out := make([]Activity, len(rows))
	for i, row := range rows {
		out[i] = Activity{Direction: row.Direction, Amount: row.Amount, When: row.When}
	}
	return out
}

func (bsvCoin) Quote(amount, unit string) (string, string, string, error) {
	return bsv.Quote(amount, unit)
}

func (bsvCoin) WarmEuro() error {
	_, err := bsv.WarmEuro()
	return err
}

func (bsvCoin) AmountFromEUR(eur string) (string, error) {
	return bsv.AmountFromEUR(eur)
}

func (bsvCoin) Send(mnemonic, to, amount string) error {
	return bsv.Send(mnemonic, to, amount)
}

func publicFrom(account cardano.Account) Public {
	return Public{Address: account.Address}
}

// registry order is the stable wallet index. Do not reorder it.
var registry = []Coin{
	cardanoCoin{},
	bsvCoin{},
}

func coinAt(index int) (Coin, error) {
	if index < 0 || index >= len(registry) {
		return nil, fmt.Errorf("unknown wallet %d", index)
	}
	return registry[index], nil
}

func coinByID(id string) (int, Coin, bool) {
	for i, c := range registry {
		if c.ID() == id {
			return i, c, true
		}
	}
	return 0, nil, false
}
