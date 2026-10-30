package cardano

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Mainnet public indexer. The balance is lovelace, shown as ADA.
var balanceEndpoint = "https://api.koios.rest/api/v1/address_info"

var balanceClient = &http.Client{Timeout: 20 * time.Second}

const balanceCacheTTL = time.Minute

type cachedBalance struct {
	lovelace string
	amount   string
	until    time.Time
}

var (
	balanceCacheMu sync.Mutex
	balanceCache   = map[string]cachedBalance{}
)

// Balance reads the mainnet amount held by address.
// A successful answer is reused for one minute so opening the wallet again
// does not repeat the request. Failures are not cached.
func Balance(address string) (string, error) {
	if amount, ok := loadCachedBalance(address); ok {
		return amount, nil
	}
	lovelace, err := fetchBalance(address)
	if err != nil {
		return "", err
	}
	amount, err := formatADA(lovelace)
	if err != nil {
		return "", fmt.Errorf("cardano: balance: %w", err)
	}
	storeCachedBalance(address, lovelace, amount)
	return amount, nil
}

func loadCachedLovelace(address string) (string, bool) {
	balanceCacheMu.Lock()
	defer balanceCacheMu.Unlock()
	entry, ok := balanceCache[address]
	if !ok || time.Now().After(entry.until) {
		return "", false
	}
	return entry.lovelace, true
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

func storeCachedBalance(address, lovelace, amount string) {
	balanceCacheMu.Lock()
	defer balanceCacheMu.Unlock()
	balanceCache[address] = cachedBalance{
		lovelace: lovelace,
		amount:   amount,
		until:    time.Now().Add(balanceCacheTTL),
	}
}

func fetchBalance(address string) (string, error) {
	payload, err := json.Marshal(map[string][]string{
		"_addresses": {address},
	})
	if err != nil {
		return "", fmt.Errorf("cardano: balance: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, balanceEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("cardano: balance: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := balanceClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cardano: balance: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("cardano: balance: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cardano: balance: %s", resp.Status)
	}
	var rows []struct {
		Balance json.RawMessage `json:"balance"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return "", fmt.Errorf("cardano: balance: %w", err)
	}
	if len(rows) == 0 {
		return "0", nil
	}
	lovelace, err := lovelaceText(rows[0].Balance)
	if err != nil {
		return "", err
	}
	return lovelace, nil
}

func lovelaceText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("cardano: balance: missing amount")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		return number.String(), nil
	}
	return "", fmt.Errorf("cardano: balance: missing amount")
}

func formatADA(lovelace string) (string, error) {
	n := new(big.Int)
	if _, ok := n.SetString(strings.TrimSpace(lovelace), 10); !ok || n.Sign() < 0 {
		return "", fmt.Errorf("bad amount %q", lovelace)
	}
	whole := new(big.Int)
	frac := new(big.Int)
	whole.QuoRem(n, big.NewInt(1_000_000), frac)
	if frac.Sign() == 0 {
		return whole.String() + " ADA", nil
	}
	digits := fmt.Sprintf("%06d", frac.Int64())
	return whole.String() + "." + strings.TrimRight(digits, "0") + " ADA", nil
}
