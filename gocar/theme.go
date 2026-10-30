package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	colBG     = color.NRGBA{R: 8, G: 10, B: 12, A: 255}
	colCard   = color.NRGBA{R: 16, G: 20, B: 24, A: 255}
	colLime   = color.NRGBA{R: 212, G: 255, B: 0, A: 255}
	colFG     = color.NRGBA{R: 232, G: 236, B: 240, A: 255}
	colMuted  = color.NRGBA{R: 148, G: 156, B: 164, A: 255}
	colBanner = color.NRGBA{R: 14, G: 40, B: 36, A: 255}
	colInk    = color.NRGBA{R: 10, G: 14, B: 10, A: 255}
	colWarn   = color.NRGBA{R: 255, G: 186, B: 64, A: 255}
	colMine   = color.NRGBA{R: 22, G: 32, B: 18, A: 255}
)

var darkBase fyne.Theme

func baseTheme() fyne.Theme {
	if darkBase == nil {
		darkBase = theme.DarkTheme()
	}
	return darkBase
}

type neonTheme struct{}

func (neonTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return colBG
	case theme.ColorNameButton:
		return colCard
	case theme.ColorNameForeground:
		return colFG
	case theme.ColorNamePrimary:
		return colLime
	case theme.ColorNameForegroundOnPrimary:
		return colInk
	case theme.ColorNameHover:
		return color.NRGBA{R: 36, G: 44, B: 32, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 180, G: 220, B: 0, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 90, G: 96, B: 90, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 32, G: 36, B: 30, A: 255}
	case theme.ColorNamePlaceHolder:
		return colMuted
	case theme.ColorNameInputBackground:
		return colCard
	case theme.ColorNameFocus, theme.ColorNameSelection:
		return colLime
	default:
		return baseTheme().Color(n, v)
	}
}

func (neonTheme) Font(style fyne.TextStyle) fyne.Resource {
	return interResource(style.Bold || style.Monospace)
}

func (neonTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return baseTheme().Icon(n)
}

func (neonTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 16
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameCaptionText:
		return 13
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 4
	case theme.SizeNameInlineIcon:
		return 22
	default:
		return baseTheme().Size(n)
	}
}
