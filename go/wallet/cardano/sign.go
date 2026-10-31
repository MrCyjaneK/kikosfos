package cardano

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/blinklabs-io/bursa"
)

// SignMessage signs msg with the CIP-1854 account key m/1854'/1815'/0'.
// The result is hex of the 64-byte signature. It is the account key itself,
// so it cannot witness a spend of a payment key derived below it.
func SignMessage(mnemonic string, msg []byte) (string, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	root, err := bursa.GetRootKeyFromMnemonic(mnemonic, "")
	if err != nil {
		return "", fmt.Errorf("cardano: %w", err)
	}
	account, err := bursa.GetMultiSigAccountKey(root, 0)
	if err != nil {
		return "", fmt.Errorf("cardano: account key: %w", err)
	}
	return hex.EncodeToString(account.Sign(msg)), nil
}
