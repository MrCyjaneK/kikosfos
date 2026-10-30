package cardano

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Activity is one confirmed payment to or from the wallet address.
type Activity struct {
	Direction string
	Amount    string
	When      string
}

const activityTimeLayout = "2 Jan 2006, 15:04"

// Newest transactions first. A later refresh only asks for rows after the
// highest block already stored, and never asks again for a transaction body
// we already have. The list itself is reused for one minute.
var (
	addressTxsEndpoint = "https://api.koios.rest/api/v1/address_txs?order=block_height.desc"
	txInfoEndpoint     = "https://api.koios.rest/api/v1/tx_info"
)

const (
	historyPageSize = 100
	historyMaxPages = 10
	historyCacheTTL = time.Minute
	txInfoBatch     = 25
)

type chainTx struct {
	hash   string
	height int64
	when   int64
	net    string
}

type historyState struct {
	txs     map[string]chainTx
	height  int64
	checked time.Time
}

var (
	historyMu    sync.Mutex
	historyCache = map[string]*historyState{}
)

// RefreshActivity loads any transactions that are not already cached.
func RefreshActivity(address string) error {
	historyMu.Lock()
	state := historyCache[address]
	if state != nil && time.Now().Before(state.checked) {
		historyMu.Unlock()
		return nil
	}
	after := int64(0)
	known := map[string]struct{}{}
	if state != nil {
		after = state.height
		for hash := range state.txs {
			known[hash] = struct{}{}
		}
	}
	historyMu.Unlock()

	listed, err := listAddressTxs(address, after)
	if err != nil {
		return err
	}
	var missing []txRef
	var maxHeight int64
	for _, ref := range listed {
		if ref.Height > maxHeight {
			maxHeight = ref.Height
		}
		if _, ok := known[ref.Hash]; ok {
			continue
		}
		missing = append(missing, ref)
	}
	details, err := txDetails(missing)
	if err != nil {
		return err
	}
	fetched := map[string]koiosTx{}
	for _, tx := range details {
		fetched[tx.Hash] = tx
	}
	pending := make(map[string]chainTx, len(missing))
	for _, ref := range missing {
		tx, ok := fetched[ref.Hash]
		if !ok {
			return fmt.Errorf("cardano: history: missing transaction")
		}
		net, err := netLovelace(address, tx)
		if err != nil {
			return fmt.Errorf("cardano: history: %w", err)
		}
		when := ref.When
		if tx.When != 0 {
			when = tx.When
		}
		height := ref.Height
		if tx.Height != 0 {
			height = tx.Height
		}
		pending[ref.Hash] = chainTx{hash: ref.Hash, height: height, when: when, net: net}
	}

	historyMu.Lock()
	defer historyMu.Unlock()
	state = historyCache[address]
	if state == nil {
		state = &historyState{txs: map[string]chainTx{}}
		historyCache[address] = state
	}
	for hash, tx := range pending {
		state.txs[hash] = tx
	}
	if maxHeight > state.height {
		state.height = maxHeight
	}
	state.checked = time.Now().Add(historyCacheTTL)
	return nil
}

// Activities is the cached list, newest first. It does not use the network.
func Activities(address string) []Activity {
	historyMu.Lock()
	state := historyCache[address]
	if state == nil || len(state.txs) == 0 {
		historyMu.Unlock()
		return nil
	}
	rows := make([]chainTx, 0, len(state.txs))
	for _, tx := range state.txs {
		rows = append(rows, tx)
	}
	historyMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].when == rows[j].when {
			return rows[i].hash > rows[j].hash
		}
		return rows[i].when > rows[j].when
	})
	out := make([]Activity, 0, len(rows))
	for _, tx := range rows {
		item, err := activityFrom(tx)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func activityFrom(tx chainTx) (Activity, error) {
	n := new(big.Int)
	if _, ok := n.SetString(tx.net, 10); !ok {
		return Activity{}, fmt.Errorf("bad amount %q", tx.net)
	}
	direction := "Sent"
	if n.Sign() > 0 {
		direction = "Received"
	}
	n.Abs(n)
	amount, err := formatADA(n.String())
	if err != nil {
		return Activity{}, err
	}
	when := ""
	if tx.when > 0 {
		when = time.Unix(tx.when, 0).Local().Format(activityTimeLayout)
	}
	return Activity{Direction: direction, Amount: amount, When: when}, nil
}

type txRef struct {
	Hash   string
	Height int64
	When   int64
}

func listAddressTxs(address string, after int64) ([]txRef, error) {
	var all []txRef
	for page := 0; page < historyMaxPages; page++ {
		start := page * historyPageSize
		chunk, err := postAddressTxs(address, after, start, start+historyPageSize-1)
		if err != nil {
			return nil, err
		}
		all = append(all, chunk...)
		if len(chunk) < historyPageSize {
			break
		}
	}
	return all, nil
}

func postAddressTxs(address string, after int64, start, end int) ([]txRef, error) {
	body := map[string]any{"_addresses": []string{address}}
	if after > 0 {
		body["_after_block_height"] = after
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, addressTxsEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Range", fmt.Sprintf("%d-%d", start, end))
	raw, err := readKoios(req)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Hash   string `json:"tx_hash"`
		Height int64  `json:"block_height"`
		When   int64  `json:"block_time"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	out := make([]txRef, 0, len(rows))
	for _, row := range rows {
		if row.Hash == "" {
			continue
		}
		out = append(out, txRef{Hash: row.Hash, Height: row.Height, When: row.When})
	}
	return out, nil
}

func txDetails(refs []txRef) ([]koiosTx, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	var all []koiosTx
	for i := 0; i < len(refs); i += txInfoBatch {
		j := i + txInfoBatch
		if j > len(refs) {
			j = len(refs)
		}
		hashes := make([]string, 0, j-i)
		for _, ref := range refs[i:j] {
			hashes = append(hashes, ref.Hash)
		}
		chunk, err := postTxInfo(hashes)
		if err != nil {
			return nil, err
		}
		all = append(all, chunk...)
	}
	return all, nil
}

type koiosAddr struct {
	Bech32 string
}

func (a *koiosAddr) UnmarshalJSON(raw []byte) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if raw[0] == '"' {
		return json.Unmarshal(raw, &a.Bech32)
	}
	var obj struct {
		Bech32 string `json:"bech32"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}
	a.Bech32 = obj.Bech32
	return nil
}

type koiosValue struct {
	Address koiosAddr
	Value   string
}

func (v *koiosValue) UnmarshalJSON(raw []byte) error {
	var row struct {
		Address koiosAddr       `json:"payment_addr"`
		Value   json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(raw, &row); err != nil {
		return err
	}
	text, err := lovelaceText(row.Value)
	if err != nil {
		return err
	}
	v.Address = row.Address
	v.Value = text
	return nil
}

type koiosTx struct {
	Hash    string       `json:"tx_hash"`
	Height  int64        `json:"block_height"`
	When    int64        `json:"block_time"`
	Inputs  []koiosValue `json:"inputs"`
	Outputs []koiosValue `json:"outputs"`
}

func postTxInfo(hashes []string) ([]koiosTx, error) {
	payload, err := json.Marshal(map[string]any{
		"_tx_hashes": hashes,
		"_inputs":    true,
	})
	if err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, txInfoEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	raw, err := readKoios(req)
	if err != nil {
		return nil, err
	}
	var rows []koiosTx
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	return rows, nil
}

func netLovelace(address string, tx koiosTx) (string, error) {
	net := new(big.Int)
	for _, row := range tx.Inputs {
		if row.Address.Bech32 != address {
			continue
		}
		value, err := parseLovelace(row.Value)
		if err != nil {
			return "", err
		}
		net.Sub(net, value)
	}
	for _, row := range tx.Outputs {
		if row.Address.Bech32 != address {
			continue
		}
		value, err := parseLovelace(row.Value)
		if err != nil {
			return "", err
		}
		net.Add(net, value)
	}
	return net.String(), nil
}

func parseLovelace(text string) (*big.Int, error) {
	n := new(big.Int)
	if _, ok := n.SetString(text, 10); !ok || n.Sign() < 0 {
		return nil, fmt.Errorf("bad amount %q", text)
	}
	return n, nil
}

func readKoios(req *http.Request) ([]byte, error) {
	resp, err := balanceClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("cardano: history: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("cardano: history: %s", resp.Status)
	}
	return body, nil
}
