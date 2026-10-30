package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// Inter (latin), SIL Open Font License 1.1, Copyright 2024 The Inter Project Authors.
//
//go:embed font/Inter-Regular.ttf
var interRegular []byte

//go:embed font/Inter-Bold.ttf
var interBold []byte

var (
	interRegRes  = fyne.NewStaticResource("Inter-Regular.ttf", interRegular)
	interBoldRes = fyne.NewStaticResource("Inter-Bold.ttf", interBold)
)

func interResource(bold bool) fyne.Resource {
	if bold {
		return interBoldRes
	}
	return interRegRes
}
