package bsv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// eurRateEndpoint overrides the Cakewallet URL. Empty builds it from Ticker.
// Tests point it at a local server.
var eurRateEndpoint string

func eurRateURL() string {
	if eurRateEndpoint != "" {
		return eurRateEndpoint
	}
	return "https://prices.cakewallet.com/v3/rates?base=crypto." + Ticker + "&quote=fiat.EUR"
}

const eurRateCacheTTL = 15 * time.Minute

var (
	eurRateMu sync.Mutex
	eurRate   struct {
		price *big.Rat
		until time.Time
	}
)

// Fiat quotes the cached balance in euros.
// The BSV price is reused for fifteen minutes. The satoshi amount comes from
// the balance cache, so this does not repeat the chain request.
func Fiat(address string) (string, error) {
	satoshis, ok := loadCachedSatoshis(address)
	if !ok {
		if _, err := Balance(address); err != nil {
			return "", err
		}
		satoshis, ok = loadCachedSatoshis(address)
		if !ok {
			return "", fmt.Errorf("bsv: balance: missing amount")
		}
	}
	price, err := eurPerBSV()
	if err != nil {
		return "", err
	}
	text, err := formatEUR(satoshis, price)
	if err != nil {
		return "", fmt.Errorf("bsv: euro price: %w", err)
	}
	return text, nil
}

func cachedEurPerBSV() (*big.Rat, bool) {
	eurRateMu.Lock()
	defer eurRateMu.Unlock()
	if eurRate.price == nil || !time.Now().Before(eurRate.until) {
		return nil, false
	}
	return new(big.Rat).Set(eurRate.price), true
}

// WarmEuro loads the BSV euro price into the cache.
func WarmEuro() (*big.Rat, error) {
	return eurPerBSV()
}

func eurPerBSV() (*big.Rat, error) {
	if price, ok := cachedEurPerBSV(); ok {
		return price, nil
	}
	price, err := fetchEurPerBSV()
	if err != nil {
		return nil, err
	}
	eurRateMu.Lock()
	eurRate.price = price
	eurRate.until = time.Now().Add(eurRateCacheTTL)
	eurRateMu.Unlock()
	return new(big.Rat).Set(price), nil
}

func fetchEurPerBSV() (*big.Rat, error) {
	req, err := http.NewRequest(http.MethodGet, eurRateURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("bsv: euro price: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := balanceClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bsv: euro price: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("bsv: euro price: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bsv: euro price: %s", resp.Status)
	}
	var payload struct {
		Results map[string]json.Number `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("bsv: euro price: %w", err)
	}
	var bestKey int64 = -1
	var best json.Number
	for key, value := range payload.Results {
		stamp, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
		if stamp > bestKey {
			bestKey = stamp
			best = value
		}
	}
	if bestKey < 0 {
		return nil, fmt.Errorf("bsv: euro price: missing rate")
	}
	price := new(big.Rat)
	if _, ok := price.SetString(best.String()); !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("bsv: euro price: bad rate")
	}
	return price, nil
}

func formatEUR(satoshis string, price *big.Rat) (string, error) {
	n := new(big.Int)
	if _, ok := n.SetString(satoshis, 10); !ok || n.Sign() < 0 {
		return "", fmt.Errorf("bad amount %q", satoshis)
	}
	eur := new(big.Rat).SetInt(n)
	eur.Quo(eur, big.NewRat(100_000_000, 1))
	eur.Mul(eur, price)
	cents := new(big.Rat).Mul(eur, big.NewRat(100, 1))
	value, _ := cents.Float64()
	rounded := int64(math.Round(value))
	if rounded == 0 && n.Sign() > 0 {
		return "< €0.01", nil
	}
	return fmt.Sprintf("€%d.%02d", rounded/100, rounded%100), nil
}

// AmountFromEUR converts a euro amount into satoshis using the BSV price.
// One euro buys 1/price BSV, and one BSV is 100_000_000 satoshis, rounded
// to the nearest satoshi.
func AmountFromEUR(text string) (string, error) {
	euros, err := parseEUR(text)
	if err != nil {
		return "", err
	}
	price, err := eurPerBSV()
	if err != nil {
		return "", err
	}
	return satoshisFromEUR(euros, price)
}

func parseEUR(text string) (*big.Rat, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "€")
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	if text == "" {
		return nil, errors.New("bsv: amount: enter a euro amount")
	}
	if strings.Contains(text, ",") {
		if strings.Contains(text, ".") {
			return nil, errors.New("bsv: amount: enter a euro amount")
		}
		text = strings.ReplaceAll(text, ",", ".")
	}
	if strings.HasPrefix(text, "-") || strings.Count(text, ".") > 1 {
		return nil, errors.New("bsv: amount: enter a euro amount")
	}
	n := new(big.Rat)
	if _, ok := n.SetString(text); !ok || n.Sign() < 0 {
		return nil, errors.New("bsv: amount: enter a euro amount")
	}
	return n, nil
}

func satoshisFromEUR(euros, price *big.Rat) (string, error) {
	if price.Sign() <= 0 {
		return "", errors.New("bsv: euro price: bad rate")
	}
	n := new(big.Rat).Quo(new(big.Rat).Set(euros), price)
	n.Mul(n, big.NewRat(100_000_000, 1))
	num := new(big.Int).Mul(n.Num(), big.NewInt(2))
	den := new(big.Int).Set(n.Denom())
	num.Add(num, den)
	den.Mul(den, big.NewInt(2))
	rounded := new(big.Int).Quo(num, den)
	if rounded.Sign() < 0 || !rounded.IsUint64() {
		return "", errors.New("bsv: amount: too large")
	}
	return rounded.String(), nil
}
