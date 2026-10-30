package wallet

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

var errIncompatible = errors.New("server is incompatible")

// discoverCurrency reads GET /api/v1/currency. The value is the coin id,
// such as "cardano".
func discoverCurrency(host string) (string, error) {
	status, body, err := kikoDo(http.MethodGet, host, "/api/v1/currency", nil, nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", errors.New(host + ": " + apiError(status, body))
	}
	var payload struct {
		Currency string `json:"currency"`
	}
	if json.Unmarshal(body, &payload) != nil || strings.TrimSpace(payload.Currency) == "" {
		return "", errors.New(host + " did not name a currency")
	}
	return payload.Currency, nil
}

func normalizeHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("enter a host")
	}
	if strings.Contains(host, "://") || strings.ContainsAny(host, "/?#@ ") {
		return "", errors.New("enter a host without a scheme or path")
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		port, err := strconv.Atoi(host[i+1:])
		if err != nil || port < 1 || port > 65535 || i == 0 || strings.Contains(host[:i], ":") {
			return "", errors.New("enter a host without a scheme or path")
		}
	}
	return host, nil
}

func hostTaken(servers []serverRecord, host string, skip int) bool {
	for i, server := range servers {
		if i == skip {
			continue
		}
		if strings.EqualFold(server.Host, host) {
			return true
		}
	}
	return false
}

func ServerCount() (int, error) {
	doc, err := loadProfile()
	if err != nil {
		return 0, err
	}
	return len(doc.Servers), nil
}

func serverAt(index int) (serverRecord, error) {
	doc, err := loadProfile()
	if err != nil {
		return serverRecord{}, err
	}
	if index < 0 || index >= len(doc.Servers) {
		return serverRecord{}, errors.New("unknown server")
	}
	return doc.Servers[index], nil
}

func ServerHost(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	return server.Host, nil
}

func ServerCurrency(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	return server.Currency, nil
}

func walletForCurrency(currency string) (int, Coin, error) {
	index, c, ok := coinByID(currency)
	if !ok {
		return 0, nil, errIncompatible
	}
	exists, err := WalletExists(index)
	if err != nil {
		return 0, nil, err
	}
	if !exists {
		return 0, nil, errIncompatible
	}
	return index, c, nil
}

func ServerCompatible(index int) (bool, error) {
	server, err := serverAt(index)
	if err != nil {
		return false, err
	}
	_, _, err = walletForCurrency(server.Currency)
	if errors.Is(err, errIncompatible) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func ServerWalletName(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	_, c, err := walletForCurrency(server.Currency)
	if errors.Is(err, errIncompatible) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return c.Name(), nil
}

func AddServer(host string) error {
	host, err := normalizeHost(host)
	if err != nil {
		return err
	}
	currency, err := discoverCurrency(host)
	if err != nil {
		return err
	}
	return updateProfile(func(doc *profileFile) error {
		if hostTaken(doc.Servers, host, -1) {
			return errors.New("server already exists")
		}
		doc.Servers = append(doc.Servers, serverRecord{Host: host, Currency: currency})
		return nil
	})
}

func UpdateServer(index int, host string) error {
	host, err := normalizeHost(host)
	if err != nil {
		return err
	}
	currency, err := discoverCurrency(host)
	if err != nil {
		return err
	}
	return updateProfile(func(doc *profileFile) error {
		if index < 0 || index >= len(doc.Servers) {
			return errors.New("unknown server")
		}
		if hostTaken(doc.Servers, host, index) {
			return errors.New("server already exists")
		}
		prev := doc.Servers[index]
		if prev.Currency != currency {
			prev.BaseFee = ""
			prev.FeePerKm = ""
		}
		doc.Servers[index] = serverRecord{
			Host:     host,
			Currency: currency,
			BaseFee:  prev.BaseFee,
			FeePerKm: prev.FeePerKm,
		}
		return nil
	})
}

// ServerCoin is the wallet index for this server's currency.
// It is -1 when the app has no such coin. The wallet does not need to exist.
func ServerCoin(index int) (int, error) {
	server, err := serverAt(index)
	if err != nil {
		return 0, err
	}
	coin, _, ok := coinByID(server.Currency)
	if !ok {
		return -1, nil
	}
	return coin, nil
}

func ServerFeeBase(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	return server.BaseFee, nil
}

func ServerFeePerKm(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	return server.FeePerKm, nil
}

// feeFromEUR converts one euro amount for a coin.
// Tests replace it so fee apply does not use the network.
var feeFromEUR = func(c Coin, eur string) (string, error) {
	return c.AmountFromEUR(eur)
}

// ApplyFeesEUR sets every server's prices from euro amounts.
// Each server uses its own currency. A failed conversion writes nothing.
func ApplyFeesEUR(baseEUR, perKmEUR string) error {
	doc, err := loadProfile()
	if err != nil {
		return err
	}
	if len(doc.Servers) == 0 {
		return errors.New("no servers")
	}
	bases := make([]string, len(doc.Servers))
	perKms := make([]string, len(doc.Servers))
	for i, server := range doc.Servers {
		_, c, ok := coinByID(server.Currency)
		if !ok {
			return errIncompatible
		}
		base, err := feeFromEUR(c, baseEUR)
		if err != nil {
			return err
		}
		perKm, err := feeFromEUR(c, perKmEUR)
		if err != nil {
			return err
		}
		bases[i] = base
		perKms[i] = perKm
	}
	return updateProfile(func(doc *profileFile) error {
		if len(doc.Servers) != len(bases) {
			return errors.New("servers changed")
		}
		for i := range doc.Servers {
			doc.Servers[i].BaseFee = bases[i]
			doc.Servers[i].FeePerKm = perKms[i]
		}
		return nil
	})
}

// SaveServerFees stores this server's prices in its currency's smallest unit.
// unit is "ada" or "lovelace". Other servers are left unchanged.
// Empty text for both prices clears them.
func SaveServerFees(index int, base, perKm, unit string) error {
	base = strings.TrimSpace(base)
	perKm = strings.TrimSpace(perKm)
	if base == "" && perKm == "" {
		return updateProfile(func(doc *profileFile) error {
			if index < 0 || index >= len(doc.Servers) {
				return errors.New("unknown server")
			}
			doc.Servers[index].BaseFee = ""
			doc.Servers[index].FeePerKm = ""
			return nil
		})
	}
	if base == "" || perKm == "" {
		return errors.New("enter both prices")
	}
	server, err := serverAt(index)
	if err != nil {
		return err
	}
	_, c, ok := coinByID(server.Currency)
	if !ok {
		return errIncompatible
	}
	_, baseLove, _, err := c.Quote(base, unit)
	if err != nil {
		return err
	}
	_, perLove, _, err := c.Quote(perKm, unit)
	if err != nil {
		return err
	}
	return updateProfile(func(doc *profileFile) error {
		if index < 0 || index >= len(doc.Servers) {
			return errors.New("unknown server")
		}
		doc.Servers[index].BaseFee = baseLove
		doc.Servers[index].FeePerKm = perLove
		return nil
	})
}

func DeleteServer(index int) error {
	return updateProfile(func(doc *profileFile) error {
		if index < 0 || index >= len(doc.Servers) {
			return errors.New("unknown server")
		}
		doc.Servers = append(doc.Servers[:index], doc.Servers[index+1:]...)
		return nil
	})
}
