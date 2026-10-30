// Package fynekit hosts a Fyne software canvas for C ABI plugins.
package fynekit

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// Kit is a windowless Fyne canvas: Capture for draw, tap/drag for touch.
type Kit struct {
	mu sync.Mutex

	canvas test.WindowlessCanvas
	last   *image.RGBA
	paused bool
	dirty  bool

	dragOn      bool
	dragObj     fyne.Draggable
	dragAbs     fyne.Position
	dragRel     fyne.Position
	dragFrom    fyne.Position
	transparent bool
	// scale is framebuffer pixels per logical unit. designH keeps type the
	// same size on every head unit.
	scale float32
}

// designH is the logical canvas height. Framebuffer height / designH is the
// software-canvas scale, matching the neon shell.
const designH = 480

func New() *Kit { return &Kit{scale: 1} }

// NewTransparent is a kit whose capture keeps empty pixels clear, so a
// caller can paint a frame underneath and composite the widgets over it.
func NewTransparent() *Kit {
	k := New()
	k.transparent = true
	return k
}

// Ensure builds the widget tree once. build runs under the kit lock.
func (k *Kit) Ensure(build func() fyne.CanvasObject) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.canvas != nil {
		return
	}
	test.NewApp()
	fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
	var c test.WindowlessCanvas
	if k.transparent {
		c = software.NewTransparentCanvas()
	} else {
		c = software.NewCanvas()
	}
	c.SetScale(1)
	c.SetPadded(true)
	c.SetContent(build())
	k.canvas = c
	k.dirty = true
}

func (k *Kit) Canvas() test.WindowlessCanvas {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.canvas
}

// MarkDirty forces the next Draw to re-Capture. Software Capture is expensive
// (one core at ~30fps); skip it when the widget tree has not changed.
func (k *Kit) MarkDirty() {
	k.mu.Lock()
	k.dirty = true
	k.mu.Unlock()
}

func (k *Kit) Draw(dst *image.RGBA) {
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return
	}

	sc := float32(h) / float32(designH)
	if sc < 0.25 {
		sc = 0.25
	}
	k.mu.Lock()
	k.scale = sc
	paused := k.paused
	dirty := k.dirty
	last := k.last
	c := k.canvas
	k.mu.Unlock()
	if c == nil {
		return
	}
	sizeOK := last != nil && last.Bounds().Dx() == w && last.Bounds().Dy() == h
	if sizeOK && (paused || !dirty) {
		draw.Draw(dst, b, last, last.Bounds().Min, draw.Src)
		return
	}

	c.SetScale(sc)
	c.Resize(fyne.NewSize(float32(w)/sc, float32(h)/sc))
	// Fyne drops unused widget renderers after 1 minute. Software Capture
	// then CreateRenderer without Layout, and Resize is a no-op at the same
	// size, so icons vanish and labels snap left while hit boxes stay.
	relayout(c.Content())
	img := c.Capture()
	blit(dst, img)

	cp := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(cp, cp.Bounds(), dst, b.Min, draw.Src)
	k.mu.Lock()
	k.last = cp
	k.dirty = false
	k.mu.Unlock()
}

// Touch maps plugin ABI (0 up, 1 down, 2 move) onto Fyne tap and drag.
func (k *Kit) Touch(x, y, action int) {
	k.mu.Lock()
	sc := k.scale
	c := k.canvas
	k.mu.Unlock()
	if c == nil {
		return
	}
	if sc < 0.05 {
		sc = 1
	}
	// Host coordinates are framebuffer pixels. Hit testing is logical.
	pos := fyne.NewPos(float32(x)/sc, float32(y)/sc)
	switch action {
	case 1:
		obj, rel := findDraggable(c, pos)
		k.mu.Lock()
		k.dragOn = true
		k.dragObj = obj
		k.dragAbs = pos
		k.dragRel = rel
		k.dragFrom = pos
		k.mu.Unlock()
		k.MarkDirty()
	case 2:
		k.mu.Lock()
		if !k.dragOn || k.dragObj == nil {
			k.mu.Unlock()
			return
		}
		dx, dy := pos.X-k.dragAbs.X, pos.Y-k.dragAbs.Y
		k.dragAbs = pos
		k.dragRel = k.dragRel.Add(fyne.NewPos(dx, dy))
		obj, rel := k.dragObj, k.dragRel
		k.mu.Unlock()
		obj.Dragged(&fyne.DragEvent{
			PointEvent: fyne.PointEvent{Position: rel, AbsolutePosition: pos},
			Dragged:    fyne.Delta{DX: dx, DY: dy},
		})
		k.MarkDirty()
	case 0:
		k.mu.Lock()
		if !k.dragOn {
			k.mu.Unlock()
			return
		}
		k.dragOn = false
		obj := k.dragObj
		from := k.dragFrom
		k.dragObj = nil
		k.mu.Unlock()
		dx, dy := pos.X-from.X, pos.Y-from.Y
		if obj != nil && dx*dx+dy*dy >= 64 {
			obj.DragEnd()
			k.MarkDirty()
			return
		}
		test.TapCanvas(c, pos)
		k.MarkDirty()
	}
}

func relayout(o fyne.CanvasObject) {
	if o == nil {
		return
	}
	sz := o.Size()
	if sz.Width < 1 || sz.Height < 1 {
		return
	}
	o.Resize(fyne.NewSize(0, 0))
	o.Resize(sz)
}

func (k *Kit) Pause() {
	k.mu.Lock()
	k.paused = true
	k.mu.Unlock()
}

func (k *Kit) Resume() {
	k.mu.Lock()
	k.paused = false
	k.dirty = true
	k.mu.Unlock()
}

func findDraggable(c fyne.Canvas, pos fyne.Position) (fyne.Draggable, fyne.Position) {
	if c == nil {
		return nil, fyne.Position{}
	}
	if ov := c.Overlays(); ov != nil {
		if top := ov.Top(); top != nil {
			if d, rel, ok := walkDraggable(top, pos, fyne.NewPos(0, 0)); ok {
				return d, rel
			}
		}
	}
	if content := c.Content(); content != nil {
		if d, rel, ok := walkDraggable(content, pos, fyne.NewPos(0, 0)); ok {
			return d, rel
		}
	}
	return nil, fyne.Position{}
}

func walkDraggable(o fyne.CanvasObject, abs, acc fyne.Position) (fyne.Draggable, fyne.Position, bool) {
	if o == nil || !o.Visible() {
		return nil, fyne.Position{}, false
	}
	pos := acc.Add(o.Position())
	sz := o.Size()
	if abs.X < pos.X || abs.Y < pos.Y || abs.X >= pos.X+sz.Width || abs.Y >= pos.Y+sz.Height {
		return nil, fyne.Position{}, false
	}
	if c, ok := o.(*fyne.Container); ok {
		for i := len(c.Objects) - 1; i >= 0; i-- {
			if d, rel, found := walkDraggable(c.Objects[i], abs, pos); found {
				return d, rel, true
			}
		}
	}
	if d, ok := o.(fyne.Draggable); ok {
		return d, abs.Subtract(pos), true
	}
	return nil, fyne.Position{}, false
}

// blit copies src into dst. Capture is usually already the framebuffer size;
// a one-pixel rounding miss from the scale is stretched.
func blit(dst *image.RGBA, src image.Image) {
	db := dst.Bounds()
	sb := src.Bounds()
	if sb.Dx() == db.Dx() && sb.Dy() == db.Dy() {
		draw.Draw(dst, db, src, sb.Min, draw.Src)
		return
	}
	if sb.Dx() < 1 || sb.Dy() < 1 || db.Dx() < 1 || db.Dy() < 1 {
		return
	}
	for y := 0; y < db.Dy(); y++ {
		sy := sb.Min.Y + y*sb.Dy()/db.Dy()
		for x := 0; x < db.Dx(); x++ {
			sx := sb.Min.X + x*sb.Dx()/db.Dx()
			r, g, b, a := src.At(sx, sy).RGBA()
			dst.SetRGBA(db.Min.X+x, db.Min.Y+y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
		}
	}
}
