package main

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

static void *kiko_open(const char *path) {
	return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static void *kiko_sym(void *h, const char *name) {
	return dlsym(h, name);
}

static const char *kiko_err(void) {
	return dlerror();
}

static void kiko_call_free(void *f, void *p) {
	((void (*)(void *))f)(p);
}

static char *kiko_call_str(void *f) {
	return ((char *(*)(void))f)();
}

static int kiko_call_init(void *f, char *a, char *b) {
	return ((int (*)(char *, char *))f)(a, b);
}

static int kiko_call_int0(void *f) {
	return ((int (*)(void))f)();
}

static int kiko_call_i_str(void *f, char *a) {
	return ((int (*)(char *))f)(a);
}

static int kiko_call_i_ss(void *f, char *a, char *b) {
	return ((int (*)(char *, char *))f)(a, b);
}

static int kiko_call_ii(void *f, int i) {
	return ((int (*)(int))f)(i);
}

static char *kiko_call_si(void *f, int i) {
	return ((char *(*)(int))f)(i);
}

static char *kiko_call_quote(void *f, int i, char *a, char *b) {
	return ((char *(*)(int, char *, char *))f)(i, a, b);
}

static char *kiko_call_refresh(void *f, double km) {
	return ((char *(*)(double))f)(km);
}

static char *kiko_call_ss(void *f, char *a, char *b) {
	return ((char *(*)(char *, char *))f)(a, b);
}

static char *kiko_call_sss(void *f, char *a, char *b, char *c) {
	return ((char *(*)(char *, char *, char *))f)(a, b, c);
}

static char *kiko_call_sssi(void *f, char *a, char *b, char *c, int i) {
	return ((char *(*)(char *, char *, char *, int))f)(a, b, c, i);
}

static unsigned char *kiko_call_icon(void *f, int i, int *n) {
	return ((unsigned char *(*)(int, int *))f)(i, n);
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

type kikoLib struct {
	mu          sync.Mutex
	handle      unsafe.Pointer
	free        unsafe.Pointer
	lastErr     unsafe.Pointer
	init        unsafe.Pointer
	pinIsSet    unsafe.Pointer
	setupPin    unsafe.Pointer
	unlock      unsafe.Pointer
	quote       unsafe.Pointer
	atom        unsafe.Pointer
	icon        unsafe.Pointer
	warm        unsafe.Pointer
	icons       map[int][]byte
	register    unsafe.Pointer
	refresh     unsafe.Pointer
	driveCount  unsafe.Pointer
	driveID     unsafe.Pointer
	driveServer unsafe.Pointer
	driveCoin   unsafe.Pointer
	driveStart  unsafe.Pointer
	driveEnd    unsafe.Pointer
	drivePickup unsafe.Pointer
	driveDist   unsafe.Pointer
	driveDay    unsafe.Pointer
	driveClock  unsafe.Pointer
	driveSLat   unsafe.Pointer
	driveSLon   unsafe.Pointer
	driveELat   unsafe.Pointer
	driveELon   unsafe.Pointer
	driveFare   unsafe.Pointer
	driveNow    unsafe.Pointer
	claim       unsafe.Pointer
	cashOut     unsafe.Pointer
	verify      unsafe.Pointer
	claimed     unsafe.Pointer
	paid        unsafe.Pointer
	activeCount unsafe.Pointer
	active      unsafe.Pointer
	navigate    unsafe.Pointer
	gps         unsafe.Pointer
}

func loadKiko() (*kikoLib, error) {
	var h unsafe.Pointer
	var last string
	for _, path := range []string{"/usr/lib64/libkiko.so", "/usr/lib/libkiko.so", "libkiko.so"} {
		cp := C.CString(path)
		h = C.kiko_open(cp)
		C.free(unsafe.Pointer(cp))
		if h != nil {
			break
		}
		if e := C.kiko_err(); e != nil {
			last = C.GoString(e)
		}
	}
	if h == nil {
		if last == "" {
			last = "libkiko.so not found"
		}
		return nil, errors.New(last)
	}
	k := &kikoLib{handle: h}
	if err := k.bind(); err != nil {
		return nil, err
	}
	return k, nil
}

func (k *kikoLib) bind() error {
	var err error
	bind := func(name string, dst *unsafe.Pointer) {
		if err != nil {
			return
		}
		var p unsafe.Pointer
		p, err = k.sym(name)
		if err == nil {
			*dst = p
		}
	}
	bind("KIKO_free", &k.free)
	bind("KIKO_last_error", &k.lastErr)
	bind("KIKO_init", &k.init)
	bind("KIKO_pin_is_set", &k.pinIsSet)
	bind("KIKO_setup_pin", &k.setupPin)
	bind("KIKO_unlock", &k.unlock)
	bind("KIKO_wallet_quote", &k.quote)
	bind("KIKO_wallet_atom", &k.atom)
	bind("KIKO_wallet_icon", &k.icon)
	bind("KIKO_wallet_warm_euro", &k.warm)
	bind("KIKO_drive_register", &k.register)
	bind("KIKO_drive_refresh", &k.refresh)
	bind("KIKO_drive_count", &k.driveCount)
	bind("KIKO_drive_id", &k.driveID)
	bind("KIKO_drive_server", &k.driveServer)
	bind("KIKO_drive_coin", &k.driveCoin)
	bind("KIKO_drive_start_address", &k.driveStart)
	bind("KIKO_drive_end_address", &k.driveEnd)
	bind("KIKO_drive_pickup_km", &k.drivePickup)
	bind("KIKO_drive_distance_km", &k.driveDist)
	bind("KIKO_drive_when_day", &k.driveDay)
	bind("KIKO_drive_when_clock", &k.driveClock)
	bind("KIKO_drive_start_lat", &k.driveSLat)
	bind("KIKO_drive_start_lon", &k.driveSLon)
	bind("KIKO_drive_end_lat", &k.driveELat)
	bind("KIKO_drive_end_lon", &k.driveELon)
	bind("KIKO_drive_fare_lovelace", &k.driveFare)
	bind("KIKO_drive_immediate", &k.driveNow)
	bind("KIKO_drive_claim", &k.claim)
	bind("KIKO_drive_cash_out", &k.cashOut)
	bind("KIKO_drive_verify", &k.verify)
	bind("KIKO_drive_claimed", &k.claimed)
	bind("KIKO_drive_paid", &k.paid)
	bind("KIKO_drive_active_count", &k.activeCount)
	bind("KIKO_drive_active", &k.active)
	bind("KIKO_drive_navigate", &k.navigate)
	bind("KIKO_gps_position", &k.gps)
	return err
}

func (k *kikoLib) sym(name string) (unsafe.Pointer, error) {
	C.kiko_err()
	cn := C.CString(name)
	p := C.kiko_sym(k.handle, cn)
	C.free(unsafe.Pointer(cn))
	if p == nil {
		msg := name + " missing"
		if e := C.kiko_err(); e != nil {
			msg = C.GoString(e)
		}
		return nil, errors.New(msg)
	}
	return p, nil
}

func (k *kikoLib) take(p *C.char) string {
	if p == nil {
		return ""
	}
	s := C.GoString(p)
	C.kiko_call_free(k.free, unsafe.Pointer(p))
	return s
}

func (k *kikoLib) LastError() string {
	return k.take(C.kiko_call_str(k.lastErr))
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func (k *kikoLib) Init(config, data string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	ca, cb := C.CString(config), C.CString(data)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	if C.kiko_call_init(k.init, ca, cb) != 0 {
		return errors.New(nonEmpty(k.LastError(), "libkiko failed"))
	}
	return nil
}

func (k *kikoLib) PinIsSet() (bool, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	n := int(C.kiko_call_int0(k.pinIsSet))
	if n < 0 {
		return false, errors.New(nonEmpty(k.LastError(), "libkiko failed"))
	}
	return n == 1, nil
}

func (k *kikoLib) SetupPin(pin, again string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	ca, cb := C.CString(pin), C.CString(again)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	if C.kiko_call_i_ss(k.setupPin, ca, cb) != 0 {
		return errors.New(nonEmpty(k.LastError(), "libkiko failed"))
	}
	return nil
}

func (k *kikoLib) Unlock(pin string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	ca := C.CString(pin)
	defer C.free(unsafe.Pointer(ca))
	if C.kiko_call_i_str(k.unlock, ca) != 0 {
		return errors.New(nonEmpty(k.LastError(), "libkiko failed"))
	}
	return nil
}

func (k *kikoLib) Quote(index int, amount string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	ca, cb := C.CString(amount), C.CString(k.atomUnit(index))
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	return k.take(C.kiko_call_quote(k.quote, C.int(index), ca, cb))
}

// atomUnit is the coin's smallest unit. Fares are stored in that unit:
// lovelace on Cardano, satoshi on BSV.
func (k *kikoLib) atomUnit(index int) string {
	if text := k.take(C.kiko_call_si(k.atom, C.int(index))); text != "" {
		return text
	}
	return "lovelace"
}

func (k *kikoLib) WarmEuro(index int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if C.kiko_call_ii(k.warm, C.int(index)) != 0 {
		return errors.New(nonEmpty(k.LastError(), "libkiko failed"))
	}
	return nil
}

// Register posts the driver. A hard failure is an error. Lines about
// individual servers are returned as text and are not a failure.
func (k *kikoLib) Register() (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	msg := k.take(C.kiko_call_str(k.register))
	if hard := k.LastError(); hard != "" {
		return "", errors.New(hard)
	}
	return msg, nil
}

func (k *kikoLib) Refresh(rangeKm float64) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	msg := k.take(C.kiko_call_refresh(k.refresh, C.double(rangeKm)))
	if msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (k *kikoLib) textAt(fn unsafe.Pointer, index int) (string, error) {
	p := C.kiko_call_si(fn, C.int(index))
	if p == nil {
		return "", errors.New(nonEmpty(k.LastError(), "unknown ride"))
	}
	return k.take(p), nil
}

func (k *kikoLib) intAt(fn unsafe.Pointer, index int) (int, error) {
	n := int(C.kiko_call_ii(fn, C.int(index)))
	if n < 0 {
		return 0, errors.New(nonEmpty(k.LastError(), "unknown ride"))
	}
	return n, nil
}

func (k *kikoLib) Offers() ([]ride, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	n := int(C.kiko_call_int0(k.driveCount))
	if n < 0 {
		return nil, errors.New(nonEmpty(k.LastError(), "ride list failed"))
	}
	out := make([]ride, 0, n)
	for i := 0; i < n; i++ {
		r, err := k.offerAt(i)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (k *kikoLib) offerAt(i int) (ride, error) {
	var r ride
	var err error
	if r.ID, err = k.textAt(k.driveID, i); err != nil {
		return r, err
	}
	if r.Server, err = k.textAt(k.driveServer, i); err != nil {
		return r, err
	}
	if r.Coin, err = k.intAt(k.driveCoin, i); err != nil {
		return r, err
	}
	if r.StartAddress, err = k.textAt(k.driveStart, i); err != nil {
		return r, err
	}
	if r.EndAddress, err = k.textAt(k.driveEnd, i); err != nil {
		return r, err
	}
	if r.PickupKm, err = k.textAt(k.drivePickup, i); err != nil {
		return r, err
	}
	if r.DistanceKm, err = k.textAt(k.driveDist, i); err != nil {
		return r, err
	}
	if r.WhenDay, err = k.textAt(k.driveDay, i); err != nil {
		return r, err
	}
	if r.WhenClock, err = k.textAt(k.driveClock, i); err != nil {
		return r, err
	}
	if r.StartLat, err = k.textAt(k.driveSLat, i); err != nil {
		return r, err
	}
	if r.StartLon, err = k.textAt(k.driveSLon, i); err != nil {
		return r, err
	}
	if r.EndLat, err = k.textAt(k.driveELat, i); err != nil {
		return r, err
	}
	if r.EndLon, err = k.textAt(k.driveELon, i); err != nil {
		return r, err
	}
	if r.FareLovelace, err = k.textAt(k.driveFare, i); err != nil {
		return r, err
	}
	if r.Immediate, err = k.intAt(k.driveNow, i); err != nil {
		return r, err
	}
	return r, nil
}

func (k *kikoLib) Active() ([]ride, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	n := int(C.kiko_call_int0(k.activeCount))
	if n < 0 {
		return nil, errors.New(nonEmpty(k.LastError(), "ride list failed"))
	}
	out := make([]ride, 0, n)
	for i := 0; i < n; i++ {
		raw, err := k.textAt(k.active, i)
		if err != nil {
			return nil, err
		}
		var r ride
		if json.Unmarshal([]byte(raw), &r) != nil {
			return nil, errors.New("bad ride")
		}
		r.Claimed = true
		r.normalize()
		out = append(out, r)
	}
	return out, nil
}

func (k *kikoLib) callSSS(fn unsafe.Pointer, a, b, c string) string {
	ca, cb, cc := C.CString(a), C.CString(b), C.CString(c)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	defer C.free(unsafe.Pointer(cc))
	return k.take(C.kiko_call_sss(fn, ca, cb, cc))
}

func (k *kikoLib) callSS(fn unsafe.Pointer, a, b string) string {
	ca, cb := C.CString(a), C.CString(b)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	return k.take(C.kiko_call_ss(fn, ca, cb))
}

func (k *kikoLib) Claim(server, id, pickupKm string, coin int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if msg := k.callSSSI(k.claim, server, id, pickupKm, coin); msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (k *kikoLib) callSSSI(fn unsafe.Pointer, a, b, c string, i int) string {
	ca, cb, cc := C.CString(a), C.CString(b), C.CString(c)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	defer C.free(unsafe.Pointer(cc))
	return k.take(C.kiko_call_sssi(fn, ca, cb, cc, C.int(i)))
}

func (k *kikoLib) CashOut(server, id string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if msg := k.callSS(k.cashOut, server, id); msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (k *kikoLib) Verify(server, id string) (bool, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	msg := k.callSS(k.verify, server, id)
	switch msg {
	case "1":
		return true, nil
	case "0":
		return false, nil
	default:
		return false, errors.New(nonEmpty(msg, "payment check failed"))
	}
}

func (k *kikoLib) Navigate(lat, lon, title string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if msg := k.callSSS(k.navigate, lat, lon, title); msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (k *kikoLib) Position() (lat, lon, acc float64, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	p := C.kiko_call_str(k.gps)
	if p == nil {
		return 0, 0, 0, errors.New(nonEmpty(k.LastError(), "Establishing GPS"))
	}
	return parseFix(k.take(p))
}

func parseFix(text string) (lat, lon, acc float64, err error) {
	parts := strings.Split(text, "\n")
	if len(parts) < 2 {
		return 0, 0, 0, errors.New("Establishing GPS")
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, 0, errors.New("Establishing GPS")
	}
	lon, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, 0, errors.New("Establishing GPS")
	}
	if len(parts) > 2 {
		acc, _ = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	}
	return lat, lon, acc, nil
}

func (k *kikoLib) decorate(rows []ride) {
	seen := map[int]struct{}{}
	for i := range rows {
		if rows[i].Coin < 0 {
			continue
		}
		if _, ok := seen[rows[i].Coin]; ok {
			continue
		}
		seen[rows[i].Coin] = struct{}{}
		_ = k.WarmEuro(rows[i].Coin)
	}
	for i := range rows {
		raw := ""
		if rows[i].Coin >= 0 && rows[i].FareLovelace != "" {
			raw = k.Quote(rows[i].Coin, rows[i].FareLovelace)
		}
		rows[i].FareAmount, rows[i].FareEuro = quoteParts(rows[i].Coin, rows[i].FareLovelace, raw)
		rows[i].Icon = k.coinIcon(rows[i].Coin)
	}
}

// coinIcon is the wallet PNG for a coin index. The bytes are kept so the
// offer list does not ask the library again on every refresh.
func (k *kikoLib) coinIcon(index int) []byte {
	if index < 0 {
		return nil
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if png, ok := k.icons[index]; ok {
		return png
	}
	if k.icons == nil {
		k.icons = map[int][]byte{}
	}
	var n C.int
	p := C.kiko_call_icon(k.icon, C.int(index), &n)
	if p == nil || n <= 0 {
		k.icons[index] = nil
		return nil
	}
	buf := C.GoBytes(unsafe.Pointer(p), n)
	C.kiko_call_free(k.free, unsafe.Pointer(p))
	k.icons[index] = buf
	return buf
}
