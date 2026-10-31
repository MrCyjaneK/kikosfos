package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"gocar/fynekit"
)

func TestFareMarkShowsCoinIcon(t *testing.T) {
	plain := fareMark("€1.20", 0, nil, 16, 22, colLime, true)
	if hasImage(plain) {
		t.Fatal("euro without an icon still drew one")
	}
	pngBytes := solidPNG(t, color.NRGBA{R: 220, G: 20, B: 60, A: 255})
	marked := fareMark("€1.20", 1, pngBytes, 16, 22, colLime, true)
	if !hasImage(marked) {
		t.Fatal("euro has no coin icon")
	}
	card := newRideCard(ride{
		Immediate:    1,
		FareEuro:     "€1.20",
		Coin:         1,
		Icon:         pngBytes,
		StartAddress: "Marienplatz",
		EndAddress:   "Hauptbahnhof",
	}, nil)
	if !hasImage(card.lines[0]) {
		t.Fatal("offer card hid the coin")
	}
}

func TestOfferIconsSitBesideEuro(t *testing.T) {
	ada, err := os.ReadFile("../go/wallet/cardano/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	bsv, err := os.ReadFile("../go/wallet/bsv/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	offers := container.New(vstack{8},
		newRideCard(ride{Immediate: 1, FareEuro: "€4.20", Coin: 0, Icon: ada, StartAddress: "Prielmayerstraße", EndAddress: "Karlsplatz"}, nil),
		newRideCard(ride{Immediate: 1, FareEuro: "€4.20", Coin: 1, Icon: bsv, StartAddress: "Prielmayerstraße", EndAddress: "Karlsplatz"}, nil),
		newRideCard(ride{WhenClock: "15:04", FareEuro: "€9.00", Coin: 1, Icon: bsv, StartAddress: "Stachus", EndAddress: "Odeonsplatz"}, nil),
	)
	k := fynekit.New()
	k.Ensure(func() fyne.CanvasObject {
		fyne.CurrentApp().Settings().SetTheme(neonTheme{})
		plate := canvas.NewRectangle(colBG)
		return container.NewStack(plate, container.NewPadded(offers))
	})
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	k.Draw(dst)
	trip := newHUD().buildTrip(snap{trip: ride{
		FareEuro: "€4.20", Coin: 1, Icon: bsv,
		StartAddress: "Prielmayerstraße", EndAddress: "Karlsplatz",
		PickupKm: "0.4", DistanceKm: "1.6", Immediate: 1,
	}})
	if !hasImage(trip) {
		t.Fatal("accept screen hid the coin")
	}
	if !hasColor(dst, func(c color.RGBA) bool {
		return c.B > 180 && c.R < 80 && c.G < 120
	}) {
		t.Fatal("cardano icon did not draw")
	}
	if !hasColor(dst, func(c color.RGBA) bool {
		return c.R > 200 && c.G > 200 && c.B > 200
	}) {
		t.Fatal("icon mark did not draw")
	}
}

func hasImage(o fyne.CanvasObject) bool {
	if o == nil {
		return false
	}
	if _, ok := o.(*canvas.Image); ok {
		return true
	}
	c, ok := o.(*fyne.Container)
	if !ok {
		return false
	}
	for _, child := range c.Objects {
		if hasImage(child) {
			return true
		}
	}
	return false
}

func hasColor(img *image.RGBA, match func(color.RGBA) bool) bool {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.RGBAAt(x, y)
			if match(c) {
				return true
			}
		}
	}
	return false
}

func solidPNG(t *testing.T, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		switch i % 4 {
		case 0:
			img.Pix[i] = c.R
		case 1:
			img.Pix[i] = c.G
		case 2:
			img.Pix[i] = c.B
		default:
			img.Pix[i] = c.A
		}
	}
	return mustPNG(t, img)
}

func mustPNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
