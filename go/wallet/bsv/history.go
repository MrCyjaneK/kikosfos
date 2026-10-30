package bsv

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

// Activity is one confirmed payment to or from the wallet address.
type Activity struct {
	Direction string
	Amount    string
	When      string
}

const activityTimeLayout = "2 Jan 2006, 15:04"

const (
	historyCacheTTL = time.Minute
	txInfoBatch     = 20
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
	known := map[string]struct{}{}
	if state != nil {
		for hash := range state.txs {
			known[hash] = struct{}{}
		}
	}
	historyMu.Unlock()

	listed, err := listHistory(address)
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
	hashes := make([]string, len(missing))
	for i, ref := range missing {
		hashes[i] = ref.Hash
	}
	bodies, err := txBodies(hashes)
	if err != nil {
		return err
	}
	decoded := map[string]*transaction.Transaction{}
	meta := map[string]txMeta{}
	for hash, body := range bodies {
		tx, err := transaction.NewTransactionFromHex(body.Hex)
		if err != nil {
			return fmt.Errorf("bsv: history: %w", err)
		}
		decoded[hash] = tx
		meta[hash] = body
	}
	if err := fillParents(decoded); err != nil {
		return err
	}
	mine, err := walletScript(address)
	if err != nil {
		return fmt.Errorf("bsv: history: %w", err)
	}
	pending := make(map[string]chainTx, len(missing))
	for _, ref := range missing {
		tx, ok := decoded[ref.Hash]
		if !ok {
			return fmt.Errorf("bsv: history: missing transaction")
		}
		net, err := netSatoshis(mine, tx, decoded)
		if err != nil {
			return err
		}
		info := meta[ref.Hash]
		height := info.Height
		if height < 1 {
			height = ref.Height
		}
		pending[ref.Hash] = chainTx{hash: ref.Hash, height: height, when: info.When, net: net}
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
	amount, err := formatBSV(n.String())
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
}

func listHistory(address string) ([]txRef, error) {
	body, err := getJSON("history", addressPath(address, "history"))
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Hash   string `json:"tx_hash"`
		Height int64  `json:"height"`
	}
	if err := decodeJSON(body, &rows); err != nil {
		return nil, fmt.Errorf("bsv: history: %w", err)
	}
	out := make([]txRef, 0, len(rows))
	for _, row := range rows {
		if row.Hash == "" || row.Height < 1 {
			continue
		}
		out = append(out, txRef{Hash: row.Hash, Height: row.Height})
	}
	return out, nil
}

type txMeta struct {
	Hex    string
	Height int64
	When   int64
}

func txBodies(hashes []string) (map[string]txMeta, error) {
	out := map[string]txMeta{}
	if len(hashes) == 0 {
		return out, nil
	}
	for i := 0; i < len(hashes); i += txInfoBatch {
		j := i + txInfoBatch
		if j > len(hashes) {
			j = len(hashes)
		}
		chunk, err := postTxs(hashes[i:j])
		if err != nil {
			return nil, err
		}
		for hash, meta := range chunk {
			out[hash] = meta
		}
	}
	return out, nil
}

func postTxs(hashes []string) (map[string]txMeta, error) {
	payload, err := json.Marshal(map[string][]string{"txids": hashes})
	if err != nil {
		return nil, fmt.Errorf("bsv: history: %w", err)
	}
	body, err := postJSON("history", apiBase+"/txs", payload)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		TxID   string `json:"txid"`
		Hash   string `json:"hash"`
		Hex    string `json:"hex"`
		Height int64  `json:"blockheight"`
		When   int64  `json:"blocktime"`
		Time   int64  `json:"time"`
	}
	if err := decodeJSON(body, &rows); err != nil {
		return nil, fmt.Errorf("bsv: history: %w", err)
	}
	out := map[string]txMeta{}
	for _, row := range rows {
		if row.Hex == "" {
			continue
		}
		when := row.When
		if when == 0 {
			when = row.Time
		}
		meta := txMeta{Hex: row.Hex, Height: row.Height, When: when}
		if row.TxID != "" {
			out[row.TxID] = meta
		}
		if row.Hash != "" {
			out[row.Hash] = meta
		}
	}
	return out, nil
}

func fillParents(decoded map[string]*transaction.Transaction) error {
	var missing []string
	seen := map[string]struct{}{}
	for _, tx := range decoded {
		for _, in := range tx.Inputs {
			if in.SourceTXID == nil {
				continue
			}
			hash := in.SourceTXID.String()
			if _, ok := decoded[hash]; ok {
				continue
			}
			if _, ok := seen[hash]; ok || zeroHash(hash) {
				continue
			}
			seen[hash] = struct{}{}
			missing = append(missing, hash)
		}
	}
	parents, err := txBodies(missing)
	if err != nil {
		return err
	}
	for hash, body := range parents {
		tx, err := transaction.NewTransactionFromHex(body.Hex)
		if err != nil {
			return fmt.Errorf("bsv: history: %w", err)
		}
		decoded[hash] = tx
	}
	return nil
}

func walletScript(address string) (*script.Script, error) {
	addr, err := script.NewAddressFromString(address)
	if err != nil {
		return nil, err
	}
	return p2pkh.Lock(addr)
}

func netSatoshis(mine *script.Script, tx *transaction.Transaction, parents map[string]*transaction.Transaction) (string, error) {
	net := new(big.Int)
	for _, out := range tx.Outputs {
		if out.LockingScript == nil || !out.LockingScript.Equals(mine) {
			continue
		}
		net.Add(net, new(big.Int).SetUint64(out.Satoshis))
	}
	for _, in := range tx.Inputs {
		if in.SourceTXID == nil {
			continue
		}
		hash := in.SourceTXID.String()
		if zeroHash(hash) {
			continue
		}
		parent := parents[hash]
		if parent == nil || int(in.SourceTxOutIndex) >= len(parent.Outputs) {
			return "", fmt.Errorf("bsv: history: missing transaction")
		}
		prev := parent.Outputs[in.SourceTxOutIndex]
		if prev.LockingScript == nil || !prev.LockingScript.Equals(mine) {
			continue
		}
		net.Sub(net, new(big.Int).SetUint64(prev.Satoshis))
	}
	return net.String(), nil
}

func zeroHash(hash string) bool {
	return strings.Trim(hash, "0") == ""
}
