package bsv

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Mainnet public indexer. Amounts are satoshis, shown as BSV.
var apiBase = "https://api.whatsonchain.com/v1/bsv/main"

var balanceClient = &http.Client{Timeout: 20 * time.Second}

const balanceCacheTTL = time.Minute

type cachedBalance struct {
	satoshis string
	amount   string
	until    time.Time
}

var (
	balanceCacheMu sync.Mutex
	balanceCache   = map[string]cachedBalance{}
)

// Balance reads the confirmed mainnet amount held by address.
// A successful answer is reused for one minute so opening the wallet again
// does not repeat the request. Failures are not cached.
func Balance(address string) (string, error) {
	if amount, ok := loadCachedBalance(address); ok {
		return amount, nil
	}
	satoshis, err := fetchBalance(address)
	if err != nil {
		return "", err
	}
	amount, err := formatBSV(satoshis)
	if err != nil {
		return "", fmt.Errorf("bsv: balance: %w", err)
	}
	storeCachedBalance(address, satoshis, amount)
	return amount, nil
}

func loadCachedSatoshis(address string) (string, bool) {
	balanceCacheMu.Lock()
	defer balanceCacheMu.Unlock()
	entry, ok := balanceCache[address]
	if !ok || time.Now().After(entry.until) {
		return "", false
	}
	return entry.satoshis, true
}

func loadCachedBalance(address string) (string, bool) {
	balanceCacheMu.Lock()
	defer balanceCacheMu.Unlock()
	entry, ok := balanceCache[address]
	if !ok || time.Now().After(entry.until) {
		return "", false
	}
	return entry.amount, true
}

func storeCachedBalance(address, satoshis, amount string) {
	balanceCacheMu.Lock()
	defer balanceCacheMu.Unlock()
	balanceCache[address] = cachedBalance{
		satoshis: satoshis,
		amount:   amount,
		until:    time.Now().Add(balanceCacheTTL),
	}
}

func fetchBalance(address string) (string, error) {
	body, err := getJSON("balance", addressPath(address, "balance"))
	if errors.Is(err, errNotFound) {
		return "0", nil
	}
	if err != nil {
		return "", err
	}
	var row struct {
		Confirmed int64 `json:"confirmed"`
	}
	if err := decodeJSON(body, &row); err != nil {
		return "", fmt.Errorf("bsv: balance: %w", err)
	}
	if row.Confirmed < 0 {
		return "", fmt.Errorf("bsv: balance: missing amount")
	}
	return fmt.Sprintf("%d", row.Confirmed), nil
}

func addressPath(address, suffix string) string {
	return apiBase + "/address/" + url.PathEscape(address) + "/" + suffix
}

func formatBSV(satoshis string) (string, error) {
	n := new(big.Int)
	if _, ok := n.SetString(strings.TrimSpace(satoshis), 10); !ok || n.Sign() < 0 {
		return "", fmt.Errorf("bad amount %q", satoshis)
	}
	whole := new(big.Int)
	frac := new(big.Int)
	whole.QuoRem(n, big.NewInt(100_000_000), frac)
	if frac.Sign() == 0 {
		return whole.String() + " BSV", nil
	}
	digits := fmt.Sprintf("%08d", frac.Int64())
	return whole.String() + "." + strings.TrimRight(digits, "0") + " BSV", nil
}
