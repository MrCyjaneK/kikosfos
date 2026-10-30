package cardano

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blinklabs-io/bursa"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// A simple payment is valid for about two hours. Mainnet slots are one second.
const ttlSlots = 7200

var (
	utxoEndpoint        = "https://api.koios.rest/api/v1/address_utxos"
	epochParamsEndpoint = "https://api.koios.rest/api/v1/epoch_params"
	tipEndpoint         = "https://api.koios.rest/api/v1/tip"
	submitEndpoint      = "https://api.koios.rest/api/v1/submittx"
)

// errNeedInput asks the selector for another ADA output. It is not shown.
var errNeedInput = errors.New("need input")

// Quote formats a typed ADA or lovelace amount without using the network.
// Euro text is empty until a price is already cached.
func Quote(amount, unit string) (string, string, string, error) {
	lovelace, err := parseAmount(amount, unit)
	if err != nil {
		return "", "", "", err
	}
	ada, err := formatADA(lovelace)
	if err != nil {
		return "", "", "", err
	}
	price, ok := cachedEurPerADA()
	if !ok {
		return ada, lovelace, "", nil
	}
	eur, err := formatEUR(lovelace, price)
	if err != nil {
		return ada, lovelace, "", nil
	}
	return ada, lovelace, eur, nil
}

// Preview is Quote, fetching the euro price when it is not cached yet.
func Preview(amount, unit string) (string, string, string, error) {
	ada, lovelace, eur, err := Quote(amount, unit)
	if err != nil || eur != "" {
		return ada, lovelace, eur, err
	}
	price, err := eurPerADA()
	if err != nil {
		return ada, lovelace, "", nil
	}
	eur, err = formatEUR(lovelace, price)
	if err != nil {
		return ada, lovelace, "", nil
	}
	return ada, lovelace, eur, nil
}

// Send builds, signs, and broadcasts a mainnet ADA payment.
// amount is an integer number of lovelace. Token outputs are left unspent.
func Send(mnemonic, to, amount string) error {
	lovelace, err := parseAmount(amount, "lovelace")
	if err != nil {
		return err
	}
	value, err := lovelaceUint(lovelace)
	if err != nil {
		return err
	}
	if value == 0 {
		return errors.New("cardano: send: enter an amount")
	}
	mnemonic = strings.TrimSpace(mnemonic)
	created, err := bursa.NewWallet(mnemonic, bursa.WithNetwork("mainnet"))
	if err != nil {
		return fmt.Errorf("cardano: send: %w", err)
	}
	from, err := paymentAddress(created.PaymentAddress)
	if err != nil {
		return err
	}
	dest, err := paymentAddress(to)
	if err != nil {
		return err
	}
	envelope, err := json.Marshal(created.PaymentExtendedSKey)
	if err != nil {
		return fmt.Errorf("cardano: send: %w", err)
	}
	signer, err := bursa.LoadKeyFromBytes(envelope)
	if err != nil {
		return fmt.Errorf("cardano: send: %w", err)
	}

	utxos, err := fetchUtxos(created.PaymentAddress)
	if err != nil {
		return err
	}
	params, coinsPerByte, err := fetchProtocol()
	if err != nil {
		return err
	}
	slot, err := fetchSlot()
	if err != nil {
		return err
	}
	minimum, err := minLovelace(dest, value, coinsPerByte)
	if err != nil {
		return err
	}
	if value < minimum {
		return errors.New("cardano: send: amount is below the minimum")
	}
	upper := slot + ttlSlots
	if upper < slot {
		upper = math.MaxUint64
	}

	sort.Slice(utxos, func(i, j int) bool { return utxos[i].value > utxos[j].value })
	var selected []utxo
	var total uint64
	for {
		signed, err := assemble(from, dest, selected, total, value, upper, coinsPerByte, signer, params)
		if err == nil {
			if err := submitTx(signed); err != nil {
				return err
			}
			forgetAddress(created.PaymentAddress)
			return nil
		}
		if !errors.Is(err, errNeedInput) {
			return err
		}
		if len(utxos) == 0 {
			return errors.New("cardano: send: not enough spendable ADA")
		}
		next := utxos[0]
		utxos = utxos[1:]
		if total > math.MaxUint64-next.value {
			return errors.New("cardano: send: amount overflow")
		}
		total += next.value
		selected = append(selected, next)
	}
}

func parseAmount(text, unit string) (string, error) {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	if text == "" {
		return "", errors.New("cardano: amount: enter a number")
	}
	// A comma is a decimal point, so a European keypad can type ADA.
	if strings.Contains(text, ",") {
		if unit != "ada" || strings.Contains(text, ".") {
			return "", errors.New("cardano: amount: enter a number")
		}
		text = strings.ReplaceAll(text, ",", ".")
	}
	var n *big.Int
	switch unit {
	case "ada":
		whole, frac, found := strings.Cut(text, ".")
		if strings.Contains(frac, ".") || strings.HasPrefix(text, "-") {
			return "", errors.New("cardano: amount: enter a number")
		}
		if !found {
			frac = ""
		}
		if whole == "" {
			whole = "0"
		}
		if !allDigits(whole) || (frac != "" && !allDigits(frac)) {
			return "", errors.New("cardano: amount: enter a number")
		}
		if len(frac) > 6 {
			return "", errors.New("cardano: amount: use at most 6 decimal places")
		}
		for len(frac) < 6 {
			frac += "0"
		}
		parsed, ok := new(big.Int).SetString(whole+frac, 10)
		if !ok {
			return "", errors.New("cardano: amount: enter a number")
		}
		n = parsed
	case "lovelace":
		if strings.Contains(text, ".") || !allDigits(text) {
			return "", errors.New("cardano: amount: lovelace is a whole number")
		}
		parsed, ok := new(big.Int).SetString(text, 10)
		if !ok {
			return "", errors.New("cardano: amount: enter a number")
		}
		n = parsed
	default:
		return "", errors.New("cardano: amount: unknown unit")
	}
	if !n.IsUint64() {
		return "", errors.New("cardano: amount: too large")
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

func lovelaceUint(text string) (uint64, error) {
	n, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, errors.New("cardano: amount: too large")
	}
	return n, nil
}

func paymentAddress(text string) (common.Address, error) {
	addr, err := common.NewAddress(strings.TrimSpace(text))
	if err != nil || addr.PaymentAddress() == nil || addr.NetworkId() != common.AddressNetworkMainnet {
		return common.Address{}, errors.New("cardano: send: address is not a mainnet payment address")
	}
	return addr, nil
}

type utxo struct {
	id    common.Blake2b256
	index uint32
	value uint64
}

func fetchUtxos(address string) ([]utxo, error) {
	payload, err := json.Marshal(map[string][]string{"_addresses": {address}})
	if err != nil {
		return nil, fmt.Errorf("cardano: send: %w", err)
	}
	body, err := koiosJSON(http.MethodPost, utxoEndpoint, payload)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Hash            string          `json:"tx_hash"`
		Index           uint32          `json:"tx_index"`
		Value           json.RawMessage `json:"value"`
		Assets          json.RawMessage `json:"asset_list"`
		DatumHash       json.RawMessage `json:"datum_hash"`
		InlineDatum     json.RawMessage `json:"inline_datum"`
		ReferenceScript json.RawMessage `json:"reference_script"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("cardano: send: %w", err)
	}
	out := make([]utxo, 0, len(rows))
	for _, row := range rows {
		// Spending these would either drop tokens or destroy a stored script.
		if jsonPresent(row.Assets) || jsonPresent(row.DatumHash) || jsonPresent(row.InlineDatum) || jsonPresent(row.ReferenceScript) {
			continue
		}
		hash, err := parseHash(row.Hash)
		if err != nil {
			return nil, err
		}
		text, err := lovelaceText(row.Value)
		if err != nil {
			return nil, fmt.Errorf("cardano: send: %w", err)
		}
		value, err := lovelaceUint(text)
		if err != nil || value == 0 {
			continue
		}
		out = append(out, utxo{id: hash, index: row.Index, value: value})
	}
	return out, nil
}

func jsonPresent(raw json.RawMessage) bool {
	text := strings.TrimSpace(string(raw))
	return text != "" && text != "null" && text != "[]" && text != "{}"
}

func parseHash(text string) (common.Blake2b256, error) {
	raw, err := hex.DecodeString(text)
	if err != nil || len(raw) != 32 {
		return common.Blake2b256{}, errors.New("cardano: send: bad utxo")
	}
	return common.NewBlake2b256Checked(raw)
}

func fetchProtocol() (bursa.ProtocolParams, uint64, error) {
	body, err := koiosJSON(http.MethodGet, epochParamsEndpoint, nil)
	if err != nil {
		return bursa.ProtocolParams{}, 0, err
	}
	var rows []struct {
		Epoch int64           `json:"epoch_no"`
		A     json.RawMessage `json:"min_fee_a"`
		B     json.RawMessage `json:"min_fee_b"`
		Coins json.RawMessage `json:"coins_per_utxo_size"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return bursa.ProtocolParams{}, 0, fmt.Errorf("cardano: send: %w", err)
	}
	bestEpoch := int64(-1)
	var params bursa.ProtocolParams
	var coins uint64
	for _, row := range rows {
		a, aErr := uintParam(row.A)
		b, bErr := uintParam(row.B)
		coin, coinErr := uintParam(row.Coins)
		if aErr != nil || bErr != nil || coinErr != nil || coin == 0 {
			continue
		}
		if row.Epoch > bestEpoch {
			bestEpoch = row.Epoch
			params = bursa.ProtocolParams{TxFeePerByte: a, TxFeeFixed: b}
			coins = coin
		}
	}
	if bestEpoch < 0 {
		return bursa.ProtocolParams{}, 0, errors.New("cardano: send: missing protocol parameters")
	}
	return params, coins, nil
}

func fetchSlot() (uint64, error) {
	body, err := koiosJSON(http.MethodGet, tipEndpoint, nil)
	if err != nil {
		return 0, err
	}
	var rows []struct {
		Slot json.RawMessage `json:"abs_slot"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, fmt.Errorf("cardano: send: %w", err)
	}
	if len(rows) == 0 {
		return 0, errors.New("cardano: send: missing chain tip")
	}
	slot, err := uintParam(rows[0].Slot)
	if err != nil {
		return 0, errors.New("cardano: send: missing chain tip")
	}
	return slot, nil
}

func uintParam(raw json.RawMessage) (uint64, error) {
	text, err := lovelaceText(raw)
	if err != nil {
		return 0, err
	}
	return lovelaceUint(text)
}

func koiosJSON(method, endpoint string, payload []byte) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("cardano: send: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := balanceClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cardano: send: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("cardano: send: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("cardano: send: %s", resp.Status)
	}
	return data, nil
}

func submitTx(raw []byte) error {
	req, err := http.NewRequest(http.MethodPost, submitEndpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("cardano: send: %w", err)
	}
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("Accept", "application/json")
	resp, err := balanceClient.Do(req)
	if err != nil {
		return fmt.Errorf("cardano: send: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("cardano: send: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		snippet := strings.TrimSpace(string(data))
		if len(snippet) > 180 {
			snippet = snippet[:180]
		}
		if snippet == "" {
			snippet = resp.Status
		}
		return fmt.Errorf("cardano: send: %s", snippet)
	}
	return nil
}

// minLovelace is coins-per-byte times the output size plus CIP-55's 160-byte overhead.
func minLovelace(addr common.Address, amount, coinsPerByte uint64) (uint64, error) {
	raw, err := (&babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: amount},
	}).MarshalCBOR()
	if err != nil {
		return 0, fmt.Errorf("cardano: send: %w", err)
	}
	size := uint64(len(raw)) + 160
	if coinsPerByte != 0 && size > math.MaxUint64/coinsPerByte {
		return 0, errors.New("cardano: send: minimum overflow")
	}
	return coinsPerByte * size, nil
}

func assemble(from, to common.Address, inputs []utxo, total, amount, upper, coinsPerByte uint64, signer *bursa.LoadedKey, params bursa.ProtocolParams) ([]byte, error) {
	fee := params.TxFeeFixed
	if fee == 0 {
		fee = 1
	}
	for range 8 {
		if total < amount || total-amount < fee {
			return nil, errNeedInput
		}
		change := total - amount - fee
		outputs := []babbage.BabbageTransactionOutput{adaOutput(to, amount)}
		if change > 0 {
			minChange, err := minLovelace(from, change, coinsPerByte)
			if err != nil {
				return nil, err
			}
			if change < minChange {
				// The leftover cannot sit in its own output, so it is paid as fee.
				fee = total - amount
				change = 0
				outputs = []babbage.BabbageTransactionOutput{adaOutput(to, amount)}
			} else {
				outputs = append(outputs, adaOutput(from, change))
			}
		}
		raw, err := paymentTx(inputs, outputs, fee, upper)
		if err != nil {
			return nil, err
		}
		signed, err := bursa.SignTransaction(raw, []*bursa.LoadedKey{signer})
		if err != nil {
			return nil, fmt.Errorf("cardano: send: %w", err)
		}
		need, err := bursa.MinFee(len(signed), params)
		if err != nil {
			return nil, fmt.Errorf("cardano: send: %w", err)
		}
		// A fee a few lovelace above the minimum is valid. Shrinking it can
		// change the transaction size and make the figure chase itself.
		if fee >= need {
			return signed, nil
		}
		fee = need
	}
	return nil, errors.New("cardano: send: fee did not settle")
}

func adaOutput(addr common.Address, amount uint64) babbage.BabbageTransactionOutput {
	return babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: amount},
	}
}

func paymentTx(inputs []utxo, outputs []babbage.BabbageTransactionOutput, fee, upper uint64) ([]byte, error) {
	bodyInputs := make([]shelley.ShelleyTransactionInput, len(inputs))
	for i, item := range inputs {
		bodyInputs[i] = shelley.ShelleyTransactionInput{TxId: item.id, OutputIndex: item.index}
	}
	body := conway.ConwayTransactionBody{
		TxInputs:  conway.NewConwayTransactionInputSet(bodyInputs),
		TxOutputs: outputs,
		TxFee:     fee,
	}
	body.SetValidityIntervalUpperBound(upper)
	tx := &conway.ConwayTransaction{
		Body:      body,
		TxIsValid: true,
	}
	raw, err := tx.MarshalCBOR()
	if err != nil {
		return nil, fmt.Errorf("cardano: send: %w", err)
	}
	return raw, nil
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
