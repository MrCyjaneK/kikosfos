package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
)

var errIncompatible = errors.New("server is incompatible")

// discoverCurrencies reads GET /api/v1/currency. The value is one coin id
// or a list of them, such as "cardano" or ["cardano", "bsv"].
func discoverCurrencies(host string) ([]string, error) {
	status, body, err := kikoDo(http.MethodGet, host, "/api/v1/currency", nil, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, errors.New(host + ": " + apiError(status, body))
	}
	var payload struct {
		Currency json.RawMessage `json:"currency"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil, errors.New(host + " did not name a currency")
	}
	raw := bytes.TrimSpace(payload.Currency)
	if len(raw) == 0 {
		return nil, errors.New(host + " did not name a currency")
	}
	var names []string
	switch raw[0] {
	case '"':
		var one string
		if json.Unmarshal(raw, &one) != nil || strings.TrimSpace(one) == "" {
			return nil, errors.New(host + " did not name a currency")
		}
		names = []string{one}
	case '[':
		if json.Unmarshal(raw, &names) != nil || len(names) == 0 {
			return nil, errors.New(host + " did not name a currency")
		}
	default:
		return nil, errors.New(host + " did not name a currency")
	}
	return names, nil
}

// chainsFor keeps the coins this app knows, in the order the server named them.
func chainsFor(names []string) ([]chainFees, error) {
	var chains []chainFees
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		if _, _, ok := coinByID(name); !ok {
			continue
		}
		seen[name] = true
		chains = append(chains, chainFees{Currency: name})
	}
	if len(chains) == 0 {
		return nil, errIncompatible
	}
	return chains, nil
}

// mergeChains keeps fees for coins the host still offers and adds an empty
// slot for a coin it newly names.
func mergeChains(prev, next []chainFees) []chainFees {
	out := make([]chainFees, len(next))
	for i, chain := range next {
		if old, ok := chainByCurrency(prev, chain.Currency); ok {
			chain.BaseFee = old.BaseFee
			chain.FeePerKm = old.FeePerKm
		}
		out[i] = chain
	}
	return out
}

func chainByCurrency(chains []chainFees, currency string) (chainFees, bool) {
	for _, chain := range chains {
		if chain.Currency == currency {
			return chain, true
		}
	}
	return chainFees{}, false
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

// canonicalHost folds the default https port into the bare name, so
// example.com and example.com:443 are one server. A loopback port stays,
// because that is a local http listener.
func canonicalHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	name, port, err := net.SplitHostPort(host)
	if err != nil || port != "443" || name == "localhost" || name == "127.0.0.1" {
		return host
	}
	return name
}

func hostTaken(servers []serverRecord, host string, skip int) bool {
	for i, server := range servers {
		if i == skip {
			continue
		}
		if canonicalHost(server.Host) == canonicalHost(host) {
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
	if len(server.Chains) == 0 {
		return "", nil
	}
	return server.Chains[0].Currency, nil
}

func firstChain(server serverRecord) chainFees {
	if len(server.Chains) == 0 {
		return chainFees{}
	}
	return server.Chains[0]
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
	for _, chain := range server.Chains {
		_, _, err = walletForCurrency(chain.Currency)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, errIncompatible) {
			return false, err
		}
	}
	return false, nil
}

func ServerWalletName(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	for _, chain := range server.Chains {
		_, c, err := walletForCurrency(chain.Currency)
		if err == nil {
			return c.Name(), nil
		}
		if !errors.Is(err, errIncompatible) {
			return "", err
		}
	}
	return "", nil
}

func AddServer(host string) error {
	host, err := normalizeHost(host)
	if err != nil {
		return err
	}
	names, err := discoverCurrencies(host)
	if err != nil {
		return err
	}
	chains, err := chainsFor(names)
	if err != nil {
		return err
	}
	return updateProfile(func(doc *profileFile) error {
		if hostTaken(doc.Servers, host, -1) {
			return errors.New("server already exists")
		}
		doc.Servers = append(doc.Servers, serverRecord{Host: host, Chains: chains})
		return nil
	})
}

func UpdateServer(index int, host string) error {
	host, err := normalizeHost(host)
	if err != nil {
		return err
	}
	names, err := discoverCurrencies(host)
	if err != nil {
		return err
	}
	chains, err := chainsFor(names)
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
		doc.Servers[index] = serverRecord{Host: host, Chains: mergeChains(prev.Chains, chains)}
		return nil
	})
}

// RefreshServers re-reads each host's currencies. Fees for a coin that remains
// are kept. A new coin gets an empty price. A coin the host no longer names
// is dropped. A host that cannot be reached keeps the chains already stored.
func RefreshServers() error {
	doc, err := loadProfile()
	if err != nil {
		return err
	}
	type change struct {
		index  int
		chains []chainFees
	}
	var changes []change
	var lines []string
	for i, server := range doc.Servers {
		names, err := discoverCurrencies(server.Host)
		if err != nil {
			lines = append(lines, err.Error())
			continue
		}
		chains, err := chainsFor(names)
		if err != nil {
			lines = append(lines, server.Host+": "+err.Error())
			continue
		}
		changes = append(changes, change{i, mergeChains(server.Chains, chains)})
	}
	if len(changes) > 0 {
		err = updateProfile(func(doc *profileFile) error {
			for _, item := range changes {
				if item.index < 0 || item.index >= len(doc.Servers) {
					return errors.New("servers changed")
				}
				doc.Servers[item.index].Chains = item.chains
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if len(lines) > 0 {
		return errors.New(strings.Join(lines, "\n"))
	}
	return nil
}

// chainView is one chain as the server page and the server list read it.
type chainView struct {
	Currency string `json:"currency"`
	Coin     int    `json:"coin"`
	Name     string `json:"name"`
	Ticker   string `json:"ticker"`
	Whole    string `json:"whole"`
	Atom     string `json:"atom"`
	Wallet   bool   `json:"wallet"`
	BaseFee  string `json:"base_fee"`
	FeePerKm string `json:"fee_per_km"`
}

func ServerChains(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	views := make([]chainView, len(server.Chains))
	for i, chain := range server.Chains {
		views[i] = viewChain(chain)
	}
	body, err := json.Marshal(views)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func viewChain(chain chainFees) chainView {
	view := chainView{
		Currency: chain.Currency,
		Coin:     -1,
		BaseFee:  chain.BaseFee,
		FeePerKm: chain.FeePerKm,
	}
	index, c, ok := coinByID(chain.Currency)
	if !ok {
		view.Name = chain.Currency
		return view
	}
	view.Coin = index
	view.Name = c.Name()
	view.Ticker = c.Ticker()
	view.Whole = c.Whole()
	view.Atom = c.Atom()
	exists, err := WalletExists(index)
	view.Wallet = err == nil && exists
	return view
}

// ServerCoin is the wallet index for this server's first chain.
// It is -1 when the app has no such coin. The wallet does not need to exist.
func ServerCoin(index int) (int, error) {
	server, err := serverAt(index)
	if err != nil {
		return 0, err
	}
	coin, _, ok := coinByID(firstChain(server).Currency)
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
	return firstChain(server).BaseFee, nil
}

func ServerFeePerKm(index int) (string, error) {
	server, err := serverAt(index)
	if err != nil {
		return "", err
	}
	return firstChain(server).FeePerKm, nil
}

// feeFromEUR converts one euro amount for a coin.
// Tests replace it so fee apply does not use the network.
var feeFromEUR = func(c Coin, eur string) (string, error) {
	return c.AmountFromEUR(eur)
}

// pricedChain is one chain's fees after a euro conversion, before they are stored.
type pricedChain struct {
	server   int
	currency string
	base     string
	perKm    string
}

// ApplyFeesEUR sets every chain's prices from euro amounts.
// A chain with no wallet is skipped. A failed conversion writes nothing.
// If every chain is skipped, nothing is written.
func ApplyFeesEUR(baseEUR, perKmEUR string) error {
	doc, err := loadProfile()
	if err != nil {
		return err
	}
	if len(doc.Servers) == 0 {
		return errors.New("no servers")
	}
	var priced []pricedChain
	var skipped int
	for i, server := range doc.Servers {
		for _, chain := range server.Chains {
			_, c, err := walletForCurrency(chain.Currency)
			if errors.Is(err, errIncompatible) {
				skipped++
				continue
			}
			if err != nil {
				return err
			}
			base, err := feeFromEUR(c, baseEUR)
			if err != nil {
				return err
			}
			perKm, err := feeFromEUR(c, perKmEUR)
			if err != nil {
				return err
			}
			priced = append(priced, pricedChain{i, chain.Currency, base, perKm})
		}
	}
	if len(priced) == 0 {
		if skipped > 0 {
			return errIncompatible
		}
		return errors.New("no servers")
	}
	return updateProfile(func(doc *profileFile) error {
		for _, item := range priced {
			if item.server < 0 || item.server >= len(doc.Servers) {
				return errors.New("servers changed")
			}
			chains := doc.Servers[item.server].Chains
			found := false
			for i := range chains {
				if chains[i].Currency == item.currency {
					chains[i].BaseFee = item.base
					chains[i].FeePerKm = item.perKm
					found = true
				}
			}
			if !found {
				return errors.New("servers changed")
			}
			doc.Servers[item.server].Chains = chains
		}
		return nil
	})
}

// SaveServerFees stores one chain's prices in its smallest unit.
// A server with one chain uses that chain. Otherwise unit selects the chain
// ("ada", "lovelace", "bsv" or "satoshi"). Empty text for both prices clears them.
func SaveServerFees(index int, base, perKm, unit string) error {
	base = strings.TrimSpace(base)
	perKm = strings.TrimSpace(perKm)
	server, err := serverAt(index)
	if err != nil {
		return err
	}
	at, c, err := chainForUnit(server, unit)
	if err != nil {
		return err
	}
	if base == "" && perKm == "" {
		return writeChainFees(index, server.Chains[at].Currency, "", "")
	}
	if base == "" || perKm == "" {
		return errors.New("enter both prices")
	}
	_, baseAtoms, _, err := c.Quote(base, unit)
	if err != nil {
		return err
	}
	_, perAtoms, _, err := c.Quote(perKm, unit)
	if err != nil {
		return err
	}
	return writeChainFees(index, server.Chains[at].Currency, baseAtoms, perAtoms)
}

type chainFeeInput struct {
	Currency string `json:"currency"`
	Base     string `json:"base"`
	PerKm    string `json:"per_km"`
	Unit     string `json:"unit"`
}

// SaveChainFees stores every editable chain on one server.
// A chain with no wallet is left as stored. Empty amounts clear that chain.
func SaveChainFees(index int, raw string) error {
	var inputs []chainFeeInput
	if json.Unmarshal([]byte(raw), &inputs) != nil {
		return errors.New("enter both prices")
	}
	server, err := serverAt(index)
	if err != nil {
		return err
	}
	type stored struct {
		currency string
		base     string
		perKm    string
	}
	var next []stored
	for _, input := range inputs {
		chain, ok := chainByCurrency(server.Chains, input.Currency)
		if !ok {
			return errors.New("unknown server")
		}
		_, c, err := walletForCurrency(chain.Currency)
		if errors.Is(err, errIncompatible) {
			continue
		}
		if err != nil {
			return err
		}
		base := strings.TrimSpace(input.Base)
		perKm := strings.TrimSpace(input.PerKm)
		if base == "" && perKm == "" {
			next = append(next, stored{chain.Currency, "", ""})
			continue
		}
		if base == "" || perKm == "" {
			return errors.New("enter both prices")
		}
		unit := strings.TrimSpace(input.Unit)
		_, baseAtoms, _, err := c.Quote(base, unit)
		if err != nil {
			return err
		}
		_, perAtoms, _, err := c.Quote(perKm, unit)
		if err != nil {
			return err
		}
		next = append(next, stored{chain.Currency, baseAtoms, perAtoms})
	}
	return updateProfile(func(doc *profileFile) error {
		if index < 0 || index >= len(doc.Servers) {
			return errors.New("unknown server")
		}
		chains := doc.Servers[index].Chains
		for _, item := range next {
			found := false
			for i := range chains {
				if chains[i].Currency == item.currency {
					chains[i].BaseFee = item.base
					chains[i].FeePerKm = item.perKm
					found = true
				}
			}
			if !found {
				return errors.New("unknown server")
			}
		}
		doc.Servers[index].Chains = chains
		return nil
	})
}

func chainForUnit(server serverRecord, unit string) (int, Coin, error) {
	if len(server.Chains) == 0 {
		return 0, nil, errIncompatible
	}
	if len(server.Chains) == 1 {
		_, c, err := walletForCurrency(server.Chains[0].Currency)
		if err != nil {
			return 0, nil, err
		}
		return 0, c, nil
	}
	unit = strings.TrimSpace(unit)
	for i, chain := range server.Chains {
		_, c, ok := coinByID(chain.Currency)
		if !ok {
			continue
		}
		if unit == c.Whole() || unit == c.Atom() {
			_, c, err := walletForCurrency(chain.Currency)
			if err != nil {
				return 0, nil, err
			}
			return i, c, nil
		}
	}
	return 0, nil, errIncompatible
}

func writeChainFees(index int, currency, base, perKm string) error {
	return updateProfile(func(doc *profileFile) error {
		if index < 0 || index >= len(doc.Servers) {
			return errors.New("unknown server")
		}
		chains := doc.Servers[index].Chains
		for i := range chains {
			if chains[i].Currency == currency {
				chains[i].BaseFee = base
				chains[i].FeePerKm = perKm
				doc.Servers[index].Chains = chains
				return nil
			}
		}
		return errors.New("unknown server")
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
