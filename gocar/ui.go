package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"gocar/fynekit"
)

func (h *hud) build() fyne.CanvasObject {
	h.statusLbl = nil
	h.navHead = nil
	h.navSub = nil
	h.navTime = nil
	h.navDist = nil
	h.drivenLbl = nil
	h.meterCap = nil
	h.payBtn = nil
	s := h.copyState()
	if !s.unlocked {
		return h.buildUnlock(s)
	}
	if s.screen == screenTrip {
		return h.buildTrip(s)
	}
	return h.buildHome(s)
}

func (h *hud) buildUnlock(s snap) fyne.CanvasObject {
	title := newText(h.heading(s), 28, colFG, true)
	title.Alignment = fyne.TextAlignCenter
	notice := widget.NewLabel(s.notice)
	notice.Wrapping = fyne.TextWrapWord
	notice.Alignment = fyne.TextAlignCenter

	keys := make([]fyne.CanvasObject, 0, 12)
	add := func(label string, primary bool, fn func()) {
		b := fynekit.NewBigButton(label, nil, fn)
		if !primary {
			b.Importance = widget.MediumImportance
		}
		if s.busy || s.pinState < 0 {
			b.Disable()
		}
		keys = append(keys, b)
	}
	for _, d := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		d := d
		add(d, false, func() { h.digit(d) })
	}
	add("Del", false, h.backspace)
	add("0", false, func() { h.digit("0") })
	add("OK", true, h.submit)

	head := container.NewVBox(title, pinMarks(len(s.pin)), notice)
	pad := container.NewGridWithColumns(3, keys...)
	return container.NewBorder(container.NewPadded(head), container.NewPadded(pad), nil, nil, spacer())
}

func (h *hud) buildHome(s snap) fyne.CanvasObject {
	title := newText("Kiko", 22, colLime, true)
	h.statusLbl = newText(oneLine(s.status, 64), 14, colMuted, false)
	refresh := newTap("Refresh", widget.LowImportance, 40, h.refresh)
	if s.busy {
		refresh.Disable()
	}
	head := container.New(vstack{2},
		container.NewBorder(nil, nil, title, refresh, spacer()),
		h.statusLbl,
	)

	var rows []fyne.CanvasObject
	if len(s.active) > 0 {
		rows = append(rows, sectionLabel("YOUR RIDE"))
		for i := range s.active {
			r := s.active[i]
			rows = append(rows, newRideCard(r, func() { h.openTrip(r) }))
		}
	}
	if len(s.offers) > 0 {
		rows = append(rows, sectionLabel("NEARBY"))
		for i := range s.offers {
			r := s.offers[i]
			rows = append(rows, newRideCard(r, func() { h.openTrip(r) }))
		}
	}
	if len(rows) == 0 {
		empty := newText("No rides nearby", 18, colMuted, true)
		empty.Alignment = fyne.TextAlignCenter
		rows = append(rows, empty)
	}
	list := container.New(vstack{6}, rows...)
	return container.NewPadded(container.NewBorder(head, nil, nil, nil, container.NewVScroll(list)))
}

func (h *hud) buildTrip(s snap) fyne.CanvasObject {
	back := newTap("Rides", widget.LowImportance, 40, h.back)
	back.Icon = theme.NavigateBackIcon()
	st := s.trip.statusText()
	if st == "" {
		st = s.trip.whenText()
	}
	badge := newText(st, 16, colLime, true)
	topRow := container.NewBorder(nil, nil, back, badge, spacer())

	head, sub, navTime, navDist := bannerParts(s.nav)
	h.navHead = newText(shorten(head, 28), 18, colLime, true)
	h.navSub = newText(shorten(sub, 40), 15, colFG, false)
	h.navTime = newText(navTime, 14, colMuted, true)
	h.navDist = newText(navDist, 14, colMuted, true)
	plate := canvas.NewRectangle(colBanner)
	plate.CornerRadius = 10
	navBox := container.NewStack(plate, container.NewPadded(container.New(vstack{4},
		container.New(hstack{10}, h.navHead, h.navSub),
		container.New(hstack{12}, h.navTime, h.navDist),
	)))

	var meter fyne.CanvasObject
	if s.trip.Claimed {
		h.drivenLbl = newText(formatKm(s.km), 48, colLime, true)
		h.drivenLbl.Alignment = fyne.TextAlignCenter
		cap, col := meterCaption(s.gpsNote, s.hasFix)
		h.meterCap = newText(cap, 13, col, true)
		h.meterCap.Alignment = fyne.TextAlignCenter
		meter = container.New(vstack{0}, h.drivenLbl, h.meterCap)
	} else {
		fare := newText(shorten(s.trip.fareText(), 18), 40, colLime, true)
		fare.Alignment = fyne.TextAlignCenter
		cap := newText("FARE", 13, colMuted, true)
		cap.Alignment = fyne.TextAlignCenter
		meter = container.New(vstack{0}, fare, cap)
	}

	away, awayUnit := measure(s.trip.PickupKm)
	tripKm, tripUnit := measure(s.trip.DistanceKm)
	third, thirdUnit, thirdCap := shorten(s.trip.fareText(), 14), "", "fare"
	if !s.trip.Claimed {
		third, thirdUnit, thirdCap = shorten(s.trip.whenText(), 14), "", "when"
	}
	stats := container.NewGridWithColumns(3,
		statCol(away, awayUnit, "away"),
		statCol(tripKm, tripUnit, "trip"),
		statCol(third, thirdUnit, thirdCap),
	)

	info := []fyne.CanvasObject{
		topRow,
		navBox,
		meter,
		routeBlock(s.trip.StartAddress, s.trip.EndAddress),
		stats,
	}
	if host := hostOnly(s.trip.Server); host != "" {
		line := newText(shorten(host, 48), 13, colMuted, false)
		line.Alignment = fyne.TextAlignCenter
		info = append(info, line)
	}
	if s.notice != "" {
		n := newText(oneLine(s.notice, 64), 15, colWarn, true)
		n.Alignment = fyne.TextAlignCenter
		info = append(info, n)
	}

	var actions []fyne.CanvasObject
	if label, show := s.trip.primary(); show {
		imp := widget.HighImportance
		text := label
		if s.trip.Claimed && s.trip.Paid == 0 {
			imp = widget.MediumImportance
			text = s.payLabel
			if text == "" {
				text = "Checking payment"
			}
		}
		b := newTap(text, imp, 52, h.primary)
		if s.trip.Claimed && s.trip.Paid == 0 {
			h.payBtn = b
		}
		if s.busy || (h.payBtn == b && s.payBusy) {
			b.Disable()
		}
		actions = append(actions, b)
	} else if s.trip.Status == "paid_out" {
		b := newTap("Paid out", widget.MediumImportance, 52, func() {})
		b.Disable()
		actions = append(actions, b)
	}
	pickup := newTap("To pickup", widget.MediumImportance, 48, func() {
		h.navigate(s.trip.StartLat, s.trip.StartLon, s.trip.StartAddress)
	})
	dest := newTap("To destination", widget.MediumImportance, 48, func() {
		h.navigate(s.trip.EndLat, s.trip.EndLon, s.trip.EndAddress)
	})
	if s.busy {
		pickup.Disable()
		dest.Disable()
	}
	actions = append(actions, container.NewGridWithColumns(2, pickup, dest))

	return container.NewPadded(container.NewBorder(
		container.New(vstack{6}, info...),
		container.New(vstack{6}, actions...),
		nil, nil,
		spacer(),
	))
}

func bannerParts(t Trip) (head, sub, time, dist string) {
	head, sub, time, dist, ok := banner(t)
	if !ok {
		if t.Err != "" {
			return t.Err, "", "", ""
		}
		return "No route", "", "", ""
	}
	if head == "" {
		head = "On route"
	}
	return head, sub, time, dist
}

func meterCaption(note string, hasFix bool) (string, color.Color) {
	if note != "" {
		return oneLine(note, 42), colWarn
	}
	if !hasFix {
		return "WAITING FOR GPS", colMuted
	}
	return "DRIVEN", colMuted
}

func formatKm(km float64) string {
	return fmt.Sprintf("%.1f km", km)
}

func measure(s string) (value, unit string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	return s, "km"
}

func pinMarks(n int) fyne.CanvasObject {
	count := n
	if count < 4 {
		count = 4
	}
	if count > 12 {
		count = 12
	}
	marks := make([]fyne.CanvasObject, count)
	for i := 0; i < count; i++ {
		marks[i] = newPinMark(i < n)
	}
	return container.NewCenter(container.New(hstack{8}, marks...))
}

type pinMark struct {
	widget.BaseWidget
	on bool
}

func newPinMark(on bool) *pinMark {
	m := &pinMark{on: on}
	m.ExtendBaseWidget(m)
	return m
}

func (m *pinMark) CreateRenderer() fyne.WidgetRenderer {
	fill := colMuted
	if m.on {
		fill = colLime
	}
	return &pinMarkRenderer{dot: canvas.NewCircle(fill)}
}

type pinMarkRenderer struct {
	dot *canvas.Circle
}

func (r *pinMarkRenderer) Destroy() {}

func (r *pinMarkRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dot}
}

func (r *pinMarkRenderer) MinSize() fyne.Size {
	return fyne.NewSize(12, 12)
}

func (r *pinMarkRenderer) Layout(sz fyne.Size) {
	r.dot.Resize(sz)
}

func (r *pinMarkRenderer) Refresh() {
	canvas.Refresh(r.dot)
}

func sectionLabel(s string) fyne.CanvasObject {
	return newText(s, 12, colMuted, true)
}

func statCol(value, unit, cap string) fyne.CanvasObject {
	n := newText(value, 18, colFG, true)
	var top fyne.CanvasObject = n
	if unit != "" {
		u := newText(unit, 12, colMuted, true)
		top = container.New(hstack{4}, n, u)
	}
	c := newText(strings.ToUpper(cap), 11, colMuted, true)
	c.Alignment = fyne.TextAlignCenter
	return container.NewCenter(container.New(vstack{1}, top, c))
}

func routeBlock(start, end string) fyne.CanvasObject {
	a := newText(shorten(strings.TrimSpace(start), 46), 16, colFG, true)
	b := newText(shorten(strings.TrimSpace(end), 46), 16, colFG, true)
	return container.New(routeLayout{}, newRouteRail(), container.New(endsLayout{}, a, b))
}

func spacer() fyne.CanvasObject {
	r := canvas.NewRectangle(color.NRGBA{})
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}

func newText(s string, size float32, c color.Color, bold bool) *canvas.Text {
	t := canvas.NewText(s, c)
	t.TextSize = size
	if bold {
		t.TextStyle = fyne.TextStyle{Bold: true}
	}
	return t
}

func setCanvas(t *canvas.Text, s string) {
	if t == nil || t.Text == s {
		return
	}
	t.Text = s
	t.Refresh()
}

func setCanvasColor(t *canvas.Text, s string, c color.Color) {
	if t == nil {
		return
	}
	if t.Text == s && t.Color == c {
		return
	}
	t.Text = s
	t.Color = c
	t.Refresh()
}

func shorten(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if n < 1 || len(r) <= n {
		return string(r)
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func oneLine(s string, n int) string {
	return shorten(strings.Join(strings.Fields(s), " "), n)
}

func hostOnly(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?"); i >= 0 {
		s = s[:i]
	}
	return s
}

func (r ride) cardTitle() string {
	if r.Claimed {
		if st := r.statusText(); st != "" {
			return st
		}
	}
	return r.whenText()
}

func cardRoute(r ride) fyne.CanvasObject {
	parts := []fyne.CanvasObject{
		newText(shorten(r.StartAddress, 18), 15, colFG, true),
		newChevron(),
		newText(shorten(r.EndAddress, 18), 15, colFG, true),
	}
	if km := strings.TrimSpace(r.PickupKm); km != "" {
		parts = append(parts, kmReadout(km))
	}
	return container.New(hstack{8}, parts...)
}

func kmReadout(km string) fyne.CanvasObject {
	n := newText(km, 14, colLime, true)
	u := newText("km", 12, colMuted, true)
	return container.New(hstack{3}, n, u)
}

type vstack struct {
	gap float32
}

func (v vstack) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	n := 0
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		m := o.MinSize()
		if m.Width > w {
			w = m.Width
		}
		h += m.Height
		n++
	}
	if n > 1 {
		h += v.gap * float32(n-1)
	}
	return fyne.NewSize(w, h)
}

func (v vstack) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		m := o.MinSize()
		o.Move(fyne.NewPos(0, y))
		o.Resize(fyne.NewSize(size.Width, m.Height))
		y += m.Height + v.gap
	}
}

type hstack struct {
	gap float32
}

func (h hstack) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, ht := float32(0), float32(0)
	n := 0
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		m := o.MinSize()
		w += m.Width
		if m.Height > ht {
			ht = m.Height
		}
		n++
	}
	if n > 1 {
		w += h.gap * float32(n-1)
	}
	return fyne.NewSize(w, ht)
}

func (h hstack) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	x := float32(0)
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		m := o.MinSize()
		o.Resize(m)
		o.Move(fyne.NewPos(x, (size.Height-m.Height)/2))
		x += m.Width + h.gap
	}
}

type endsLayout struct{}

func (endsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	n := 0
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		m := o.MinSize()
		if m.Width > w {
			w = m.Width
		}
		h += m.Height
		n++
	}
	if n > 1 {
		h += 14
	}
	return fyne.NewSize(w, h)
}

func (endsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	shown := make([]fyne.CanvasObject, 0, len(objects))
	for _, o := range objects {
		if o != nil && o.Visible() {
			shown = append(shown, o)
		}
	}
	if len(shown) == 0 {
		return
	}
	if len(shown) == 1 {
		m := shown[0].MinSize()
		shown[0].Resize(fyne.NewSize(size.Width, m.Height))
		shown[0].Move(fyne.NewPos(0, 0))
		return
	}
	top, bot := shown[0], shown[len(shown)-1]
	mt, mb := top.MinSize(), bot.MinSize()
	top.Resize(fyne.NewSize(size.Width, mt.Height))
	top.Move(fyne.NewPos(0, 0))
	bot.Resize(fyne.NewSize(size.Width, mb.Height))
	bot.Move(fyne.NewPos(0, size.Height-mb.Height))
}

type routeLayout struct{}

func (routeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		m := o.MinSize()
		w += m.Width
		if m.Height > h {
			h = m.Height
		}
	}
	if w > 0 {
		w += 10
	}
	return fyne.NewSize(w, h)
}

func (routeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	shown := make([]fyne.CanvasObject, 0, len(objects))
	for _, o := range objects {
		if o != nil && o.Visible() {
			shown = append(shown, o)
		}
	}
	if len(shown) == 0 {
		return
	}
	rail := shown[0]
	rw := rail.MinSize().Width
	rail.Move(fyne.NewPos(0, 0))
	rail.Resize(fyne.NewSize(rw, size.Height))
	if len(shown) < 2 {
		return
	}
	body := shown[1]
	body.Move(fyne.NewPos(rw+10, 0))
	bw := size.Width - rw - 10
	if bw < 1 {
		bw = 1
	}
	body.Resize(fyne.NewSize(bw, size.Height))
}

type chevron struct {
	widget.BaseWidget
}

func newChevron() *chevron {
	c := &chevron{}
	c.ExtendBaseWidget(c)
	return c
}

func (c *chevron) CreateRenderer() fyne.WidgetRenderer {
	a := canvas.NewLine(colMuted)
	b := canvas.NewLine(colMuted)
	a.StrokeWidth = 2
	b.StrokeWidth = 2
	return &chevRenderer{a: a, b: b}
}

type chevRenderer struct {
	a, b *canvas.Line
}

func (r *chevRenderer) Destroy() {}

func (r *chevRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.a, r.b}
}

func (r *chevRenderer) MinSize() fyne.Size {
	return fyne.NewSize(12, 12)
}

func (r *chevRenderer) Layout(sz fyne.Size) {
	tip := fyne.NewPos(sz.Width-2, sz.Height/2)
	r.a.Position1 = fyne.NewPos(2, 2)
	r.a.Position2 = tip
	r.b.Position1 = tip
	r.b.Position2 = fyne.NewPos(2, sz.Height-2)
}

func (r *chevRenderer) Refresh() {
	r.a.StrokeColor = colMuted
	r.b.StrokeColor = colMuted
	canvas.Refresh(r.a)
	canvas.Refresh(r.b)
}

type routeRail struct {
	widget.BaseWidget
}

func newRouteRail() *routeRail {
	r := &routeRail{}
	r.ExtendBaseWidget(r)
	return r
}

func (r *routeRail) CreateRenderer() fyne.WidgetRenderer {
	dot := canvas.NewCircle(colLime)
	box := canvas.NewRectangle(colLime)
	box.CornerRadius = 1
	stem := canvas.NewRectangle(color.NRGBA{R: 212, G: 255, B: 0, A: 120})
	return &railRenderer{dot: dot, box: box, stem: stem}
}

type railRenderer struct {
	dot  *canvas.Circle
	box  *canvas.Rectangle
	stem *canvas.Rectangle
}

func (r *railRenderer) Destroy() {}

func (r *railRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.stem, r.dot, r.box}
}

func (r *railRenderer) MinSize() fyne.Size {
	return fyne.NewSize(14, 36)
}

func (r *railRenderer) Layout(sz fyne.Size) {
	const mark = float32(10)
	r.dot.Resize(fyne.NewSize(mark, mark))
	r.dot.Move(fyne.NewPos((sz.Width-mark)/2, 1))
	r.box.Resize(fyne.NewSize(mark, mark))
	r.box.Move(fyne.NewPos((sz.Width-mark)/2, sz.Height-mark-1))
	stemH := sz.Height - 2*mark - 6
	if stemH < 1 {
		stemH = 1
	}
	r.stem.Resize(fyne.NewSize(2, stemH))
	r.stem.Move(fyne.NewPos((sz.Width-2)/2, mark+3))
}

func (r *railRenderer) Refresh() {
	r.dot.FillColor = colLime
	r.box.FillColor = colLime
	canvas.Refresh(r.dot)
	canvas.Refresh(r.box)
	canvas.Refresh(r.stem)
}

type tapButton struct {
	widget.Button
	tall float32
}

func newTap(label string, imp widget.Importance, tall float32, fn func()) *tapButton {
	b := &tapButton{tall: tall}
	b.Text = label
	b.OnTapped = fn
	b.Importance = imp
	b.ExtendBaseWidget(b)
	return b
}

func (b *tapButton) MinSize() fyne.Size {
	b.ExtendBaseWidget(b)
	m := b.BaseWidget.MinSize()
	if m.Height < b.tall {
		m.Height = b.tall
	}
	return m
}

type rideCard struct {
	widget.BaseWidget
	lines  []fyne.CanvasObject
	accent bool
	tap    func()
}

func newRideCard(r ride, tap func()) *rideCard {
	c := &rideCard{accent: r.Claimed, tap: tap}
	c.lines = []fyne.CanvasObject{
		splitLine(r.cardTitle(), shorten(r.fareText(), 14), colLime, colLime, 16),
		cardRoute(r),
	}
	c.ExtendBaseWidget(c)
	return c
}

func splitLine(left, right string, lc, rc color.Color, size float32) fyne.CanvasObject {
	l := newText(left, size, lc, true)
	rt := newText(right, size, rc, true)
	rt.Alignment = fyne.TextAlignTrailing
	return container.NewBorder(nil, nil, l, rt, spacer())
}

func (c *rideCard) Tapped(*fyne.PointEvent) {
	if c.tap != nil {
		c.tap()
	}
}

func (c *rideCard) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(colCard)
	bg.CornerRadius = 10
	if c.accent {
		bg.FillColor = colMine
	}
	var bar *canvas.Rectangle
	if c.accent {
		bar = canvas.NewRectangle(colLime)
		bar.CornerRadius = 2
	}
	return &cardRenderer{card: c, bg: bg, bar: bar, lines: c.lines}
}

type cardRenderer struct {
	card  *rideCard
	bg    *canvas.Rectangle
	bar   *canvas.Rectangle
	lines []fyne.CanvasObject
}

func (r *cardRenderer) Destroy() {}

func (r *cardRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.bg}
	if r.bar != nil {
		objs = append(objs, r.bar)
	}
	return append(objs, r.lines...)
}

func (r *cardRenderer) MinSize() fyne.Size {
	const padX, padY, gap = float32(12), float32(6), float32(2)
	h := padY * 2
	w := float32(0)
	for i, o := range r.lines {
		m := o.MinSize()
		if m.Width > w {
			w = m.Width
		}
		h += m.Height
		if i > 0 {
			h += gap
		}
	}
	extra := padX
	if r.bar != nil {
		extra += 6
	}
	return fyne.NewSize(w+padX+extra, h)
}

func (r *cardRenderer) Layout(sz fyne.Size) {
	const padX, padY, gap = float32(12), float32(6), float32(2)
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(sz)
	x := padX
	if r.bar != nil {
		r.bar.Move(fyne.NewPos(0, 6))
		r.bar.Resize(fyne.NewSize(4, sz.Height-12))
		x = 16
	}
	innerW := sz.Width - x - padX
	if innerW < 1 {
		innerW = 1
	}
	y := padY
	for _, o := range r.lines {
		m := o.MinSize()
		o.Move(fyne.NewPos(x, y))
		o.Resize(fyne.NewSize(innerW, m.Height))
		y += m.Height + gap
	}
}

func (r *cardRenderer) Refresh() {
	if r.card.accent {
		r.bg.FillColor = colMine
	} else {
		r.bg.FillColor = colCard
	}
	canvas.Refresh(r.card)
}
