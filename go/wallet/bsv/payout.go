package bsv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
)

// secondSigBytes is the room left for the signature the server adds:
// a push opcode and a DER signature of at most 72 bytes plus the sighash byte.
const secondSigBytes = 74

// loadScriptUTXOs reads confirmed outputs locked by a script. Tests replace it.
var loadScriptUTXOs = fetchScriptUTXOs

// Payout spends every confirmed output locked by scriptHex and pays the
// driver's own address, minus a fee that covers the server's signature too.
// index is m/48'/236'/0'/0/index. The result is Extended Format hex. Each
// input is unlocked by OP_0 and the driver's sighash. It is not broadcast.
func Payout(mnemonic, scriptHex string, index uint32) (string, error) {
	tx, err := buildPayout(mnemonic, scriptHex, index, nil)
	if err != nil {
		return "", err
	}
	ef, err := tx.EFHex()
	if err != nil {
		return "", fmt.Errorf("bsv: payout: %w", err)
	}
	return ef, nil
}

func buildPayout(mnemonic, scriptHex string, index uint32, inputs []utxo) (*transaction.Transaction, error) {
	scriptHex = strings.TrimSpace(scriptHex)
	if scriptHex == "" {
		return nil, errors.New("bsv: payout: ride has no script")
	}
	if _, err := script.NewFromHex(scriptHex); err != nil {
		return nil, errors.New("bsv: payout: bad script")
	}
	account, err := Restore(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("bsv: payout: %w", err)
	}
	dest, _, err := mainnetAddress(account.Address)
	if err != nil {
		return nil, err
	}
	priv, err := driverPaymentKey(mnemonic, index)
	if err != nil {
		return nil, fmt.Errorf("bsv: payout: %w", err)
	}
	if inputs == nil {
		inputs, err = loadScriptUTXOs(scriptHex)
		if err != nil {
			return nil, err
		}
	}
	if len(inputs) == 0 {
		return nil, errors.New("bsv: payout: ride has no funds")
	}
	var total uint64
	for i := range inputs {
		if inputs[i].script == "" {
			inputs[i].script = scriptHex
		}
		total += inputs[i].value
	}
	return sweepRide(inputs, total, dest, priv)
}

func sweepRide(inputs []utxo, total uint64, dest string, priv *ec.PrivateKey) (*transaction.Transaction, error) {
	fee := uint64(1)
	for attempt := 0; attempt < 8; attempt++ {
		if total <= fee {
			return nil, errors.New("bsv: payout: not enough for the fee")
		}
		tx := transaction.NewTransaction()
		for _, in := range inputs {
			if err := tx.AddInputFrom(in.hash, in.index, in.script, in.value, nil); err != nil {
				return nil, fmt.Errorf("bsv: payout: %w", err)
			}
		}
		if err := tx.PayToAddress(dest, total-fee); err != nil {
			return nil, fmt.Errorf("bsv: payout: %w", err)
		}
		if err := signRideInputs(tx, priv); err != nil {
			return nil, err
		}
		want := uint64(tx.Size()) + uint64(len(tx.Inputs))*secondSigBytes
		want *= feePerByte
		if want < 1 {
			want = 1
		}
		if fee >= want {
			return tx, nil
		}
		fee = want
	}
	return nil, errors.New("bsv: payout: fee did not settle")
}

func signRideInputs(tx *transaction.Transaction, priv *ec.PrivateKey) error {
	for i := range tx.Inputs {
		hash, err := tx.CalcInputSignatureHash(uint32(i), sighash.AllForkID)
		if err != nil {
			return fmt.Errorf("bsv: payout: %w", err)
		}
		sig, err := priv.Sign(hash)
		if err != nil {
			return fmt.Errorf("bsv: payout: %w", err)
		}
		raw := append(sig.Serialize(), byte(sighash.AllForkID))
		unlock := &script.Script{}
		if err := unlock.AppendOpcodes(script.Op0); err != nil {
			return fmt.Errorf("bsv: payout: %w", err)
		}
		if err := unlock.AppendPushData(raw); err != nil {
			return fmt.Errorf("bsv: payout: %w", err)
		}
		tx.Inputs[i].UnlockingScript = unlock
	}
	return nil
}

func fetchScriptUTXOs(scriptHex string) ([]utxo, error) {
	hash, err := scriptHash(scriptHex)
	if err != nil {
		return nil, err
	}
	body, err := getJSON("payout", apiBase+"/script/"+hash+"/confirmed/unspent")
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := decodeScriptUTXOs(body)
	if err != nil {
		return nil, fmt.Errorf("bsv: payout: %w", err)
	}
	out := make([]utxo, 0, len(rows))
	for _, row := range rows {
		if row.Height < 1 || row.Value == 0 || row.TxHash == "" {
			continue
		}
		out = append(out, utxo{hash: row.TxHash, script: scriptHex, index: uint32(row.TxPos), value: uint64(row.Value)})
	}
	return out, nil
}

type scriptUTXO struct {
	Height int64  `json:"height"`
	TxHash string `json:"tx_hash"`
	TxPos  int64  `json:"tx_pos"`
	Value  int64  `json:"value"`
}

func decodeScriptUTXOs(body []byte) ([]scriptUTXO, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var rows []scriptUTXO
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, err
		}
		return rows, nil
	}
	var envelope struct {
		Result []scriptUTXO `json:"result"`
		Error  string       `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != "" {
		return nil, errors.New(envelope.Error)
	}
	return envelope.Result, nil
}

// scriptHash is the sha256 of the script, reversed, which is how the indexer names it.
func scriptHash(scriptHex string) (string, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(scriptHex))
	if err != nil || len(raw) == 0 {
		return "", errors.New("bsv: payout: bad script")
	}
	sum := sha256.Sum256(raw)
	for i, j := 0, len(sum)-1; i < j; i, j = i+1, j-1 {
		sum[i], sum[j] = sum[j], sum[i]
	}
	return hex.EncodeToString(sum[:]), nil
}
