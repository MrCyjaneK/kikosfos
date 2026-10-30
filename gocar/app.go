package main

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"gocar/fynekit"
)

const (
	screenUnlock = iota
	screenHome
	screenTrip

	// Home registers and reloads rides on this interval while a refresh
	// is not already running. The trip screen does not.
	refreshEvery = 10
	// Unpaid claimed rides re-check the fare on this interval.
	payEvery = 4
)

type hud struct {
	kit *fynekit.Kit
	ui  sync.Mutex
	mu  sync.Mutex
	syn sync.Mutex

	once   sync.Once
	opened bool

	api        *kikoLib
	unlocked   bool
	screen     int
	pinState   int // -1 unavailable, 0 unset, 1 set
	confirm    bool
	first      string
	pin        string
	notice     string
	busy       bool
	refreshing bool
	nextIn     int
	status     string
	active     []ride
	offers     []ride
	listKey    string
	trip       ride
	nav        Trip
	odom       odometer

	dirtyTree bool
	dirtyLive bool
	gpsNote   string
	payLabel  string
	payBusy   bool

	statusLbl  *canvas.Text
	refreshBtn *tapButton
	navHead    *canvas.Text
	navSub     *canvas.Text
	navTime    *canvas.Text
	navDist    *canvas.Text
	drivenLbl  *canvas.Text
	meterCap   *canvas.Text
	payBtn     *tapButton
}

func newHUD() *hud {
	return &hud{kit: fynekit.New(), pinState: -1, nextIn: refreshEvery}
}

var gHUD = newHUD()

func (h *hud) start() {
	h.once.Do(func() { go h.loop() })
}

func (h *hud) loop() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	n := 0
	for range tick.C {
		n++
		h.pollTrip()
		h.tickRefresh()
		if n%payEvery == 0 && h.onTripUnpaid() {
			go h.pollPayment()
		}
	}
}

func (h *hud) onHome() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.unlocked && h.screen == screenHome && h.api != nil
}

func (h *hud) onTripUnpaid() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.unlocked && h.screen == screenTrip && h.api != nil && h.trip.Claimed && h.trip.Paid == 0
}

func (h *hud) connect() {
	lib, err := loadKiko()
	if err != nil {
		h.failConnect(err)
		return
	}
	cfg, data := kikoDirs()
	if err = lib.Init(cfg, data); err != nil {
		h.mu.Lock()
		h.api = lib
		h.mu.Unlock()
		h.failConnect(err)
		return
	}
	set, err := lib.PinIsSet()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.api = lib
	if err != nil {
		h.pinState = -1
		h.notice = err.Error()
		h.dirtyTree = true
		return
	}
	if set {
		h.pinState = 1
	} else {
		h.pinState = 0
	}
	h.notice = ""
	h.dirtyTree = true
}

func (h *hud) failConnect(err error) {
	h.mu.Lock()
	h.pinState = -1
	h.notice = err.Error()
	h.dirtyTree = true
	h.mu.Unlock()
}

func (h *hud) digit(d string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busy || h.pinState < 0 || h.unlocked {
		return
	}
	if len(h.pin) >= 12 {
		return
	}
	h.pin += d
	h.dirtyTree = true
}

func (h *hud) backspace() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busy || h.pin == "" {
		return
	}
	h.pin = h.pin[:len(h.pin)-1]
	h.dirtyTree = true
}

func (h *hud) submit() {
	h.mu.Lock()
	if h.busy || h.pin == "" || h.pinState < 0 || h.api == nil || h.unlocked {
		h.mu.Unlock()
		return
	}
	if h.pinState == 0 && !h.confirm {
		h.first = h.pin
		h.pin = ""
		h.confirm = true
		h.notice = ""
		h.dirtyTree = true
		h.mu.Unlock()
		return
	}
	pin, first := h.pin, h.first
	setup := h.pinState == 0
	api := h.api
	h.busy = true
	h.notice = ""
	h.dirtyTree = true
	h.mu.Unlock()
	go func() {
		var err error
		if setup {
			err = api.SetupPin(first, pin)
		} else {
			err = api.Unlock(pin)
		}
		h.mu.Lock()
		h.busy = false
		h.pin = ""
		if err != nil {
			h.first = ""
			h.confirm = false
			h.notice = err.Error()
			h.dirtyTree = true
			h.mu.Unlock()
			return
		}
		h.unlocked = true
		h.screen = screenHome
		h.status = "Looking for rides"
		h.notice = ""
		h.confirm = false
		h.refreshing = true
		h.dirtyTree = true
		h.mu.Unlock()
		h.syncDrives(true)
		h.endRefresh()
	}()
}

func (h *hud) refresh() {
	h.mu.Lock()
	if h.refreshing || !h.unlocked || h.api == nil {
		h.mu.Unlock()
		return
	}
	h.refreshing = true
	h.dirtyLive = true
	h.mu.Unlock()
	go h.finishRefresh()
}

func (h *hud) tickRefresh() {
	h.mu.Lock()
	if !h.unlocked || h.screen != screenHome || h.api == nil {
		h.mu.Unlock()
		return
	}
	if h.refreshing {
		h.dirtyLive = true
		h.mu.Unlock()
		return
	}
	if h.nextIn > 0 {
		h.nextIn--
	}
	due := h.nextIn <= 0
	if due {
		h.refreshing = true
	}
	h.dirtyLive = true
	h.mu.Unlock()
	if due {
		go h.finishRefresh()
	}
}

func (h *hud) finishRefresh() {
	h.syncDrives(true)
	h.endRefresh()
}

func (h *hud) endRefresh() {
	h.mu.Lock()
	h.refreshing = false
	h.nextIn = refreshEvery
	if h.screen == screenHome {
		h.dirtyLive = true
	}
	h.mu.Unlock()
}

// syncDrives reloads offers. register posts the driver first; the
// timer refresh skips that and only asks for rides.
func (h *hud) syncDrives(register bool) {
	h.syn.Lock()
	defer h.syn.Unlock()
	h.syncBody(register)
}

// syncIfIdle drops a timer tick when a refresh is already in flight.
func (h *hud) syncIfIdle() {
	if !h.syn.TryLock() {
		return
	}
	defer h.syn.Unlock()
	h.syncBody(false)
}

func (h *hud) syncBody(register bool) {
	h.mu.Lock()
	api := h.api
	h.mu.Unlock()
	if api == nil {
		return
	}
	hint := ""
	if register {
		regMsg, regErr := api.Register()
		if regErr != nil {
			hint = regErr.Error()
		} else if regMsg != "" {
			hint = regMsg
		}
	}
	if err := api.Refresh(5); err != nil {
		if hint == "" {
			hint = err.Error()
		} else {
			hint = hint + "\n" + err.Error()
		}
	}
	active, aerr := api.Active()
	offers, oerr := api.Offers()
	if aerr != nil && hint == "" {
		hint = aerr.Error()
	}
	if oerr != nil && hint == "" {
		hint = oerr.Error()
	}
	if aerr != nil {
		active = nil
	}
	if oerr != nil {
		offers = nil
	}
	rows := append(append([]ride{}, active...), offers...)
	api.decorate(rows)
	nAct := len(active)
	if nAct > len(rows) {
		nAct = len(rows)
	}
	active = rows[:nAct]
	offers = rows[nAct:]
	if hint == "" && len(active) == 0 && len(offers) == 0 {
		hint = "No rides"
	}
	key := ridesKey(active, offers)
	h.mu.Lock()
	h.active = active
	h.offers = offers
	h.status = hint
	changed := key != h.listKey
	h.listKey = key
	if h.screen == screenHome && (changed || h.dirtyTree) {
		h.dirtyTree = true
	} else if h.screen == screenHome {
		h.dirtyLive = true
	}
	h.mu.Unlock()
}

func (h *hud) openTrip(r ride) {
	r.normalize()
	h.mu.Lock()
	h.trip = r
	h.screen = screenTrip
	h.notice = ""
	h.nav = Trip{}
	h.gpsNote = ""
	h.payBusy = false
	h.payLabel = ""
	if r.Claimed && r.Paid == 0 {
		h.payLabel = "Checking payment"
	}
	h.odom.enter(r.Claimed)
	h.dirtyTree = true
	h.mu.Unlock()
	go h.pollTrip()
	if r.Claimed && r.Paid == 0 {
		go h.pollPayment()
	}
}

func (h *hud) back() {
	h.mu.Lock()
	h.screen = screenHome
	h.odom.leave()
	h.gpsNote = ""
	h.notice = ""
	h.payLabel = ""
	h.dirtyTree = true
	h.mu.Unlock()
	go h.syncIfIdle()
}

func (h *hud) primary() {
	h.mu.Lock()
	if h.busy || h.api == nil || h.screen != screenTrip {
		h.mu.Unlock()
		return
	}
	trip := h.trip
	api := h.api
	label, ok := trip.primary()
	if !ok {
		h.mu.Unlock()
		return
	}
	if label == "Check payment" {
		h.mu.Unlock()
		go h.pollPayment()
		return
	}
	h.busy = true
	h.notice = ""
	h.dirtyTree = true
	h.mu.Unlock()
	go func() {
		var err error
		switch label {
		case "Accept":
			err = api.Claim(trip.Server, trip.ID, trip.PickupKm)
		case "Cash out":
			err = api.CashOut(trip.Server, trip.ID)
		}
		h.mu.Lock()
		h.busy = false
		if h.trip.ID != trip.ID {
			h.dirtyTree = true
			h.mu.Unlock()
			return
		}
		if err != nil {
			h.notice = err.Error()
		} else {
			switch label {
			case "Accept":
				h.trip.Claimed = true
				h.trip.Status = "accepted"
				h.notice = ""
				h.payLabel = "Checking payment"
				h.odom.enter(true)
			case "Cash out":
				h.trip.Status = "paid_out"
				h.notice = ""
			}
		}
		h.dirtyTree = true
		h.mu.Unlock()
		if err == nil && label == "Accept" {
			h.syncDrives(false)
			h.pollPayment()
		}
	}()
}

func (h *hud) pollPayment() {
	h.mu.Lock()
	if h.screen != screenTrip || h.api == nil || !h.trip.Claimed || h.trip.Paid != 0 || h.payBusy || h.busy {
		h.mu.Unlock()
		return
	}
	api := h.api
	id := h.trip.ID
	server := h.trip.Server
	h.payBusy = true
	h.payLabel = "Checking payment"
	h.dirtyLive = true
	h.mu.Unlock()

	paid, err := api.Verify(server, id)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.payBusy = false
	if h.screen != screenTrip || h.trip.ID != id {
		h.dirtyTree = true
		return
	}
	if err != nil {
		h.payLabel = "Check failed"
		h.notice = err.Error()
		h.dirtyTree = true
		return
	}
	hadNotice := h.notice != ""
	h.notice = ""
	if paid {
		h.trip.Paid = 1
		h.trip.Status = "paid"
		h.payLabel = ""
		h.dirtyTree = true
		return
	}
	h.payLabel = "Unpaid"
	if hadNotice {
		h.dirtyTree = true
		return
	}
	h.dirtyLive = true
}

func (h *hud) navigate(lat, lon, title string) {
	h.mu.Lock()
	if h.busy || h.api == nil {
		h.mu.Unlock()
		return
	}
	api := h.api
	h.busy = true
	h.notice = ""
	h.dirtyTree = true
	h.mu.Unlock()
	go func() {
		err := api.Navigate(lat, lon, title)
		h.mu.Lock()
		h.busy = false
		if err != nil {
			h.notice = err.Error()
		}
		h.dirtyTree = true
		h.mu.Unlock()
	}()
}

func (h *hud) pollTrip() {
	h.mu.Lock()
	if h.screen != screenTrip || h.api == nil {
		h.mu.Unlock()
		return
	}
	api := h.api
	claimed := h.odom.live
	id := h.trip.ID
	h.mu.Unlock()

	tr := pollPure()
	var lat, lon, acc float64
	ok := false
	gpsNote := ""
	if claimed {
		var err error
		lat, lon, acc, err = api.Position()
		ok = err == nil
		if err != nil {
			gpsNote = err.Error()
		}
	}
	h.mu.Lock()
	if h.screen == screenTrip && h.trip.ID == id {
		h.nav = tr
		h.gpsNote = gpsNote
		h.odom.sample(lat, lon, acc, ok)
		h.dirtyLive = true
	}
	h.mu.Unlock()
}

type snap struct {
	screen     int
	pinState   int
	confirm    bool
	pin        string
	notice     string
	busy       bool
	status     string
	refreshing bool
	nextIn     int
	active     []ride
	offers     []ride
	trip       ride
	nav        Trip
	km         float64
	gpsNote    string
	hasFix     bool
	payLabel   string
	payBusy    bool
	unlocked   bool
}

func (h *hud) copyState() snap {
	h.mu.Lock()
	defer h.mu.Unlock()
	return snap{
		screen:     h.screen,
		pinState:   h.pinState,
		confirm:    h.confirm,
		pin:        h.pin,
		notice:     h.notice,
		busy:       h.busy,
		status:     h.status,
		refreshing: h.refreshing,
		nextIn:     h.nextIn,
		active:     append([]ride(nil), h.active...),
		offers:     append([]ride(nil), h.offers...),
		trip:       h.trip,
		nav:        h.nav,
		km:         h.odom.km,
		gpsNote:    h.gpsNote,
		hasFix:     h.odom.has,
		payLabel:   h.payLabel,
		payBusy:    h.payBusy,
		unlocked:   h.unlocked,
	}
}

func (h *hud) ensure() {
	h.mu.Lock()
	first := !h.opened
	if first {
		h.opened = true
	}
	h.mu.Unlock()
	if first {
		h.connect()
	}
	h.kit.Ensure(func() fyne.CanvasObject {
		fyne.CurrentApp().Settings().SetTheme(neonTheme{})
		return h.build()
	})
}

func (h *hud) flush() {
	h.mu.Lock()
	tree := h.dirtyTree
	live := h.dirtyLive
	h.dirtyTree = false
	h.dirtyLive = false
	nav := h.nav
	km := h.odom.km
	status := h.status
	gpsNote := h.gpsNote
	hasFix := h.odom.has
	payLabel := h.payLabel
	payBusy := h.payBusy
	busy := h.busy
	refreshing := h.refreshing
	nextIn := h.nextIn
	h.mu.Unlock()
	if h.kit.Canvas() == nil {
		if tree || live {
			h.mu.Lock()
			h.dirtyTree = h.dirtyTree || tree
			h.dirtyLive = h.dirtyLive || live
			h.mu.Unlock()
		}
		return
	}
	if tree {
		h.kit.Canvas().SetContent(h.build())
	}
	if tree || live {
		h.paintLive(nav, km, status, gpsNote, hasFix, payLabel, payBusy, busy, refreshing, nextIn)
		h.kit.MarkDirty()
	}
}

func (h *hud) paintLive(nav Trip, km float64, status, gpsNote string, hasFix bool, payLabel string, payBusy, busy bool, refreshing bool, nextIn int) {
	setCanvas(h.statusLbl, oneLine(status, 64))
	head, sub, time, dist := bannerParts(nav)
	setCanvas(h.navHead, shorten(head, 28))
	setCanvas(h.navSub, shorten(sub, 40))
	setCanvas(h.navTime, time)
	setCanvas(h.navDist, dist)
	setCanvas(h.drivenLbl, formatKm(km))
	cap, col := meterCaption(gpsNote, hasFix)
	setCanvasColor(h.meterCap, cap, col)
	if h.refreshBtn != nil {
		label := refreshLabel(refreshing, nextIn)
		if h.refreshBtn.Text != label {
			h.refreshBtn.SetText(label)
		}
		if refreshing && !h.refreshBtn.Disabled() {
			h.refreshBtn.Disable()
		} else if !refreshing && h.refreshBtn.Disabled() {
			h.refreshBtn.Enable()
		}
	}
	if h.payBtn == nil || payLabel == "" {
		return
	}
	if h.payBtn.Text != payLabel {
		h.payBtn.SetText(payLabel)
	}
	if payBusy || busy {
		if !h.payBtn.Disabled() {
			h.payBtn.Disable()
		}
		return
	}
	if h.payBtn.Disabled() {
		h.payBtn.Enable()
	}
}

func (h *hud) heading(s snap) string {
	if s.pinState < 0 {
		return "Wallet unavailable"
	}
	if s.pinState == 0 && !s.confirm {
		return "Choose a PIN"
	}
	if s.confirm {
		return "Repeat the PIN"
	}
	return "Enter PIN"
}
