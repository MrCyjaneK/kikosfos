package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"sync"
	"unsafe"

	"kiko-sfos/wallet"
)

var (
	errMu   sync.Mutex
	lastErr string
)

func setErr(err error) {
	errMu.Lock()
	defer errMu.Unlock()
	if err == nil {
		lastErr = ""
		return
	}
	lastErr = err.Error()
}

func cString(s string) *C.char {
	return C.CString(s)
}

func cBytes(b []byte, length *C.int) *C.uchar {
	if length != nil {
		*length = C.int(len(b))
	}
	if len(b) == 0 {
		return nil
	}
	return (*C.uchar)(C.CBytes(b))
}

//export KIKO_free
func KIKO_free(p unsafe.Pointer) {
	C.free(p)
}

//export KIKO_last_error
func KIKO_last_error() *C.char {
	errMu.Lock()
	msg := lastErr
	errMu.Unlock()
	return C.CString(msg)
}

//export KIKO_init
func KIKO_init(configDir, dataDir *C.char) C.int {
	err := wallet.Init(C.GoString(configDir), C.GoString(dataDir))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_pin_is_set
func KIKO_pin_is_set() C.int {
	set, err := wallet.PinIsSet()
	setErr(err)
	if err != nil {
		return -1
	}
	if set {
		return 1
	}
	return 0
}

//export KIKO_setup_pin
func KIKO_setup_pin(pin, again *C.char) C.int {
	err := wallet.SetupPin(C.GoString(pin), C.GoString(again))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_unlock
func KIKO_unlock(pin *C.char) C.int {
	err := wallet.Unlock(C.GoString(pin))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_wallet_count
func KIKO_wallet_count() C.int {
	setErr(nil)
	return C.int(wallet.WalletCount())
}

//export KIKO_wallet_name
func KIKO_wallet_name(index C.int) *C.char {
	name, err := wallet.WalletName(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(name)
}

//export KIKO_wallet_icon
func KIKO_wallet_icon(index C.int, length *C.int) *C.uchar {
	icon, err := wallet.WalletIcon(int(index))
	setErr(err)
	if err != nil {
		if length != nil {
			*length = 0
		}
		return nil
	}
	return cBytes(icon, length)
}

//export KIKO_wallet_exists
func KIKO_wallet_exists(index C.int) C.int {
	ok, err := wallet.WalletExists(int(index))
	setErr(err)
	if err != nil {
		return -1
	}
	if ok {
		return 1
	}
	return 0
}

//export KIKO_wallet_address
func KIKO_wallet_address(index C.int) *C.char {
	address, err := wallet.WalletAddress(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(address)
}

//export KIKO_wallet_detail_count
func KIKO_wallet_detail_count(index C.int) C.int {
	n, err := wallet.WalletDetailCount(int(index))
	setErr(err)
	if err != nil {
		return -1
	}
	return C.int(n)
}

//export KIKO_wallet_detail_label
func KIKO_wallet_detail_label(index, detail C.int) *C.char {
	label, err := wallet.WalletDetailLabel(int(index), int(detail))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(label)
}

//export KIKO_wallet_detail_value
func KIKO_wallet_detail_value(index, detail C.int) *C.char {
	value, err := wallet.WalletDetailValue(int(index), int(detail))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(value)
}

//export KIKO_wallet_qr
func KIKO_wallet_qr(index C.int, length *C.int) *C.uchar {
	png, err := wallet.WalletQR(int(index))
	setErr(err)
	if err != nil {
		if length != nil {
			*length = 0
		}
		return nil
	}
	return cBytes(png, length)
}

//export KIKO_wallet_create
func KIKO_wallet_create(index C.int) C.int {
	err := wallet.WalletCreate(int(index))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_wallet_restore
func KIKO_wallet_restore(index C.int, mnemonic *C.char) C.int {
	err := wallet.WalletRestore(int(index), C.GoString(mnemonic))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_wallet_mnemonic
func KIKO_wallet_mnemonic(index C.int) *C.char {
	mnemonic, err := wallet.WalletMnemonic(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(mnemonic)
}

//export KIKO_wallet_private_key
func KIKO_wallet_private_key(index C.int) *C.char {
	key, err := wallet.WalletPrivateKey(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(key)
}

//export KIKO_wallet_multisig_material
func KIKO_wallet_multisig_material(index C.int) *C.char {
	key, err := wallet.WalletMultisigMaterial(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(key)
}

//export KIKO_wallet_balance
func KIKO_wallet_balance(index C.int) *C.char {
	amount, err := wallet.WalletBalance(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(amount)
}

//export KIKO_wallet_fiat
func KIKO_wallet_fiat(index C.int) *C.char {
	amount, err := wallet.WalletFiat(int(index))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(amount)
}

//export KIKO_wallet_activity
func KIKO_wallet_activity(index C.int) C.int {
	err := wallet.WalletRefreshActivity(int(index))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_wallet_activity_count
func KIKO_wallet_activity_count(index C.int) C.int {
	n, err := wallet.WalletActivityCount(int(index))
	setErr(err)
	if err != nil {
		return -1
	}
	return C.int(n)
}

//export KIKO_wallet_activity_direction
func KIKO_wallet_activity_direction(index, row C.int) *C.char {
	text, err := wallet.WalletActivityDirection(int(index), int(row))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(text)
}

//export KIKO_wallet_activity_amount
func KIKO_wallet_activity_amount(index, row C.int) *C.char {
	text, err := wallet.WalletActivityAmount(int(index), int(row))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(text)
}

//export KIKO_wallet_activity_when
func KIKO_wallet_activity_when(index, row C.int) *C.char {
	text, err := wallet.WalletActivityWhen(int(index), int(row))
	setErr(err)
	if err != nil {
		return nil
	}
	return cString(text)
}

//export KIKO_wallet_delete
func KIKO_wallet_delete(index C.int, pin *C.char) C.int {
	err := wallet.WalletDelete(int(index), C.GoString(pin))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_wallet_quote
func KIKO_wallet_quote(index C.int, amount, unit *C.char) *C.char {
	ada, lovelace, eur, err := wallet.WalletQuote(int(index), C.GoString(amount), C.GoString(unit))
	if err != nil {
		return cString("\n\n\n" + err.Error())
	}
	return cString(ada + "\n" + lovelace + "\n" + eur + "\n")
}

//export KIKO_wallet_warm_euro
func KIKO_wallet_warm_euro(index C.int) C.int {
	err := wallet.WalletWarmEuro(int(index))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

//export KIKO_wallet_send
func KIKO_wallet_send(index C.int, to, amount *C.char) C.int {
	err := wallet.WalletSend(int(index), C.GoString(to), C.GoString(amount))
	setErr(err)
	if err != nil {
		return -1
	}
	return 0
}

func main() {}
