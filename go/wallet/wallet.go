// Package wallet creates and restores one Shelley account.
//
// The account is the default CIP-1852 wallet used by Lace, Yoroi, and Eternl:
// payment key m/1852'/1815'/0'/0/0 and stake key m/1852'/1815'/0'/2/0.
// Address is the base address for that pair, which is the address that receives ADA.
// The BIP39 passphrase is empty. A non-empty passphrase would derive a different account.
package wallet

import (
	"fmt"
	"strings"

	"github.com/blinklabs-io/bursa"
)

// Network is a Cardano network name.
type Network string

const (
	Mainnet Network = "mainnet"
	Preprod Network = "preprod"
	Preview Network = "preview"
)

// Wallet is the restored or newly generated account.
// Mnemonic is the 12-, 15-, or 24-word recovery phrase. Generate always returns 24 words.
type Wallet struct {
	Mnemonic     string
	Address      string
	StakeAddress string
}

// Generate creates a new 24-word account on network.
func Generate(network Network) (Wallet, error) {
	mnemonic, err := bursa.GenerateMnemonic()
	if err != nil {
		return Wallet{}, fmt.Errorf("wallet: generate mnemonic: %w", err)
	}
	return fromMnemonic(mnemonic, network)
}

// Restore rebuilds the account from a BIP39 recovery phrase.
// mnemonic may be 12, 15, or 24 words. Surrounding whitespace is ignored.
func Restore(mnemonic string, network Network) (Wallet, error) {
	return fromMnemonic(strings.TrimSpace(mnemonic), network)
}

func fromMnemonic(mnemonic string, network Network) (Wallet, error) {
	created, err := bursa.NewWallet(mnemonic, bursa.WithNetwork(string(network)))
	if err != nil {
		return Wallet{}, fmt.Errorf("wallet: %w", err)
	}
	return Wallet{
		Mnemonic:     mnemonic,
		Address:      created.PaymentAddress,
		StakeAddress: created.StakeAddress,
	}, nil
}
