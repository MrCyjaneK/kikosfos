package bsv

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

// One satoshi per byte. A zero-satoshi change output is left out and paid as fee.
const feePerByte = 1

var errNeedInput = errors.New("need input")

// Quote formats a typed BSV or satoshi amount without using the network.
// Euro text is empty until a price is already cached.
func Quote(amount, unit string) (string, string, string, error) {
	satoshis, err := parseAmount(amount, unit)
	if err != nil {
		return "", "", "", err
	}
	text, err := formatBSV(satoshis)
	if err != nil {
		return "", "", "", err
	}
	price, ok := cachedEurPerBSV()
	if !ok {
		return text, satoshis, "", nil
	}
	eur, err := formatEUR(satoshis, price)
	if err != nil {
		return text, satoshis, "", nil
	}
	return text, satoshis, eur, nil
}

// Send builds, signs, and broadcasts a mainnet BSV payment.
// amount is an integer number of satoshis. Change returns to the same address.
func Send(mnemonic, to, amount string) error {
	satoshis, err := parseAmount(amount, Atom)
	if err != nil {
		return err
	}
	value, err := satoshiUint(satoshis)
	if err != nil {
		return err
	}
	if value == 0 {
		return errors.New("bsv: send: enter an amount")
	}
	account, err := Restore(mnemonic)
	if err != nil {
		return fmt.Errorf("bsv: send: %w", err)
	}
	from, fromScript, err := mainnetAddress(account.Address)
	if err != nil {
		return err
	}
	dest, _, err := mainnetAddress(to)
	if err != nil {
		return err
	}
	priv, err := ec.PrivateKeyFromWif(account.PrivateKey)
	if err != nil {
		return fmt.Errorf("bsv: send: %w", err)
	}
	unlocker, err := p2pkh.Unlock(priv, nil)
	if err != nil {
		return fmt.Errorf("bsv: send: %w", err)
	}

	utxos, err := fetchUtxos(from, fromScript.String())
	if err != nil {
		return err
	}
	sort.Slice(utxos, func(i, j int) bool { return utxos[i].value > utxos[j].value })
	var selected []utxo
	var total uint64
	for {
		signed, err := settle(from, dest, selected, total, value, unlocker)
		if err == nil {
			if err := submitTx(signed.Hex()); err != nil {
				return err
			}
			forgetAddress(from)
			return nil
		}
		if !errors.Is(err, errNeedInput) {
			return err
		}
		if len(utxos) == 0 {
			return errors.New("bsv: send: not enough spendable BSV")
		}
		next := utxos[0]
		utxos = utxos[1:]
		if total > math.MaxUint64-next.value {
			return errors.New("bsv: send: amount overflow")
		}
		total += next.value
		selected = append(selected, next)
	}
}

func settle(from, to string, inputs []utxo, total, amount uint64, unlocker *p2pkh.P2PKH) (*transaction.Transaction, error) {
	fee := uint64(feePerByte)
	for range 8 {
		signed, paid, err := assemble(from, to, inputs, total, amount, fee, unlocker)
		if err != nil {
			return nil, err
		}
		need := uint64(signed.Size())
		if need > 0 && paid >= need {
			return signed, nil
		}
		if need == 0 {
			return nil, errors.New("bsv: send: fee did not settle")
		}
		fee = need
	}
	return nil, errors.New("bsv: send: fee did not settle")
}

func parseAmount(text, unit string) (string, error) {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	if text == "" {
		return "", errors.New("bsv: amount: enter a number")
	}
	if strings.Contains(text, ",") {
		if unit != Whole || strings.Contains(text, ".") {
			return "", errors.New("bsv: amount: enter a number")
		}
		text = strings.ReplaceAll(text, ",", ".")
	}
	var n *big.Int
	switch unit {
	case Whole:
		whole, frac, found := strings.Cut(text, ".")
		if strings.Contains(frac, ".") || strings.HasPrefix(text, "-") {
			return "", errors.New("bsv: amount: enter a number")
		}
		if !found {
			frac = ""
		}
		if whole == "" {
			whole = "0"
		}
		if !allDigits(whole) || (frac != "" && !allDigits(frac)) {
			return "", errors.New("bsv: amount: enter a number")
		}
		if len(frac) > 8 {
			return "", errors.New("bsv: amount: use at most 8 decimal places")
		}
		for len(frac) < 8 {
			frac += "0"
		}
		parsed, ok := new(big.Int).SetString(whole+frac, 10)
		if !ok {
			return "", errors.New("bsv: amount: enter a number")
		}
		n = parsed
	case Atom:
		if strings.Contains(text, ".") || !allDigits(text) {
			return "", errors.New("bsv: amount: satoshi is a whole number")
		}
		parsed, ok := new(big.Int).SetString(text, 10)
		if !ok {
			return "", errors.New("bsv: amount: enter a number")
		}
		n = parsed
	default:
		return "", errors.New("bsv: amount: unknown unit")
	}
	if !n.IsUint64() {
		return "", errors.New("bsv: amount: too large")
	}
	return strconv.FormatUint(n.Uint64(), 10), nil
}

func allDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func satoshiUint(text string) (uint64, error) {
	n, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, errors.New("bsv: amount: too large")
	}
	return n, nil
}

func mainnetAddress(text string) (string, *script.Script, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "1") {
		return "", nil, errors.New("bsv: send: address is not a mainnet payment address")
	}
	addr, err := script.NewAddressFromString(text)
	if err != nil {
		return "", nil, errors.New("bsv: send: address is not a mainnet payment address")
	}
	lock, err := p2pkh.Lock(addr)
	if err != nil {
		return "", nil, errors.New("bsv: send: address is not a mainnet payment address")
	}
	return addr.AddressString, lock, nil
}

type utxo struct {
	hash   string
	script string
	index  uint32
	value  uint64
}

func fetchUtxos(address, scriptHex string) ([]utxo, error) {
	body, err := getJSON("send", addressPath(address, "unspent"))
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Height int64  `json:"height"`
		Index  uint32 `json:"tx_pos"`
		Hash   string `json:"tx_hash"`
		Value  uint64 `json:"value"`
		Script string `json:"script"`
	}
	if err := decodeJSON(body, &rows); err != nil {
		return nil, fmt.Errorf("bsv: send: %w", err)
	}
	out := make([]utxo, 0, len(rows))
	for _, row := range rows {
		// Height 0 is still in the mempool. Spend confirmed outputs only.
		if row.Height < 1 || row.Value == 0 || row.Hash == "" {
			continue
		}
		lock := row.Script
		if lock == "" {
			lock = scriptHex
		}
		out = append(out, utxo{hash: row.Hash, script: lock, index: row.Index, value: row.Value})
	}
	return out, nil
}

func assemble(from, to string, inputs []utxo, total, amount, fee uint64, unlocker *p2pkh.P2PKH) (*transaction.Transaction, uint64, error) {
	if len(inputs) == 0 || total < amount || total-amount < fee {
		return nil, 0, errNeedInput
	}
	change := total - amount - fee
	paid := fee
	tx := transaction.NewTransaction()
	for _, in := range inputs {
		if err := tx.AddInputFrom(in.hash, in.index, in.script, in.value, unlocker); err != nil {
			return nil, 0, fmt.Errorf("bsv: send: %w", err)
		}
	}
	if err := tx.PayToAddress(to, amount); err != nil {
		return nil, 0, fmt.Errorf("bsv: send: %w", err)
	}
	if change >= 1 {
		if err := tx.PayToAddress(from, change); err != nil {
			return nil, 0, fmt.Errorf("bsv: send: %w", err)
		}
	} else {
		paid = total - amount
	}
	if err := tx.Sign(); err != nil {
		return nil, 0, fmt.Errorf("bsv: send: %w", err)
	}
	return tx, paid, nil
}

func submitTx(raw string) error {
	payload, err := json.Marshal(map[string]string{"txhex": raw})
	if err != nil {
		return fmt.Errorf("bsv: send: %w", err)
	}
	if _, err := postJSON("send", apiBase+"/tx/raw", payload); err != nil {
		return err
	}
	return nil
}

func forgetAddress(address string) {
	balanceCacheMu.Lock()
	delete(balanceCache, address)
	balanceCacheMu.Unlock()
	historyMu.Lock()
	if state := historyCache[address]; state != nil {
		state.checked = time.Time{}
	}
	historyMu.Unlock()
}
