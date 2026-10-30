package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	_ "image/png"
)

//go:embed icon.png
var iconPNG []byte

var (
	iconPix      []byte
	iconW, iconH int
)

func loadIcon() {
	if iconPix != nil {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	iconPix = dst.Pix
	iconW = dst.Bounds().Dx()
	iconH = dst.Bounds().Dy()
}
