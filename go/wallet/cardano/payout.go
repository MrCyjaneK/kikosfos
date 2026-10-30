package cardano

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/blinklabs-io/bursa"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// serverWitnessBytes is the room left for the signature the server adds.
// The posted transaction carries only the driver's witness. The server keeps
// the body and appends its own, so the fee has to cover that larger
// transaction. 110 matches the witness size the server itself reserves.
const serverWitnessBytes = 110

// Payout spends every plain-ADA output at payTo and pays the driver's own
// address, minus the fee. scriptHex is the ride native script. index is the
// CIP-1854 payment key that must sign it. The result is the hex of the
// driver-signed transaction, ready for the server to cosign. It is not
// submitted from here.
func Payout(mnemonic, payTo, scriptHex string, index uint32) (string, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	payTo = strings.TrimSpace(payTo)
	if payTo == "" {
		return "", errors.New("cardano: payout: ride has no address")
	}
	script, err := decodeScript(scriptHex)
	if err != nil {
		return "", err
	}
	created, err := bursa.NewWallet(mnemonic, bursa.WithNetwork("mainnet"))
	if err != nil {
		return "", fmt.Errorf("cardano: payout: %w", err)
	}
	dest, err := paymentAddress(created.PaymentAddress)
	if err != nil {
		return "", err
	}
	signer, err := multisigSigner(mnemonic, index)
	if err != nil {
		return "", err
	}
	utxos, err := fetchUtxos(payTo)
	if err != nil {
		return "", err
	}
	if len(utxos) == 0 {
		return "", errors.New("cardano: payout: ride address has no spendable ADA")
	}
	sort.Slice(utxos, func(i, j int) bool {
		if utxos[i].id != utxos[j].id {
			return utxos[i].id.String() < utxos[j].id.String()
		}
		return utxos[i].index < utxos[j].index
	})
	var total uint64
	for _, item := range utxos {
		if total > ^uint64(0)-item.value {
			return "", errors.New("cardano: payout: amount overflow")
		}
		total += item.value
	}
	params, coinsPerByte, err := fetchProtocol()
	if err != nil {
		return "", err
	}
	slot, err := fetchSlot()
	if err != nil {
		return "", err
	}
	upper := slot + ttlSlots
	if upper < slot {
		upper = ^uint64(0)
	}
	fee := params.TxFeeFixed
	if fee == 0 {
		fee = 1
	}
	for range 8 {
		if total <= fee {
			return "", errors.New("cardano: payout: ride address does not cover the fee")
		}
		value := total - fee
		minimum, err := minLovelace(dest, value, coinsPerByte)
		if err != nil {
			return "", err
		}
		if value < minimum {
			return "", errors.New("cardano: payout: ride address does not cover the fee")
		}
		raw, err := scriptPaymentTx(utxos, []babbage.BabbageTransactionOutput{adaOutput(dest, value)}, fee, upper, script)
		if err != nil {
			return "", err
		}
		signed, err := bursa.SignTransaction(raw, []*bursa.LoadedKey{signer})
		if err != nil {
			return "", fmt.Errorf("cardano: payout: %w", err)
		}
		need, err := bursa.MinFee(len(signed)+serverWitnessBytes, params)
		if err != nil {
			return "", fmt.Errorf("cardano: payout: %w", err)
		}
		if fee >= need {
			return hex.EncodeToString(signed), nil
		}
		fee = need
	}
	return "", errors.New("cardano: payout: fee did not settle")
}

func decodeScript(scriptHex string) (common.NativeScript, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(scriptHex))
	if err != nil || len(raw) == 0 {
		return common.NativeScript{}, errors.New("cardano: payout: bad script")
	}
	var script common.NativeScript
	if _, err := cbor.Decode(raw, &script); err != nil || len(script.Cbor()) == 0 {
		return common.NativeScript{}, errors.New("cardano: payout: bad script")
	}
	return script, nil
}

func multisigSigner(mnemonic string, index uint32) (*bursa.LoadedKey, error) {
	root, err := bursa.GetRootKeyFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	account, err := bursa.GetMultiSigAccountKey(root, 0)
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	payment, err := bursa.GetMultiSigPaymentKey(account, index)
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	file, err := bursa.GetMultiSigPaymentSKey(payment)
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	signer, err := bursa.LoadKeyFromBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	return signer, nil
}

func scriptPaymentTx(inputs []utxo, outputs []babbage.BabbageTransactionOutput, fee, upper uint64, script common.NativeScript) ([]byte, error) {
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
		Body: body,
		WitnessSet: conway.ConwayTransactionWitnessSet{
			WsNativeScripts: cbor.NewSetType([]common.NativeScript{script}, true),
		},
		TxIsValid: true,
	}
	raw, err := tx.MarshalCBOR()
	if err != nil {
		return nil, fmt.Errorf("cardano: payout: %w", err)
	}
	return raw, nil
}
