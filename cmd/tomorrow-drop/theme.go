package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// darkTheme is the dark reference theme (PRD §19.2): neutral colors with
// discreet purple accents and success or error indicators.
type darkTheme struct{}

var (
	colBackground = color.NRGBA{R: 0x16, G: 0x14, B: 0x1B, A: 0xFF}
	colSurface    = color.NRGBA{R: 0x21, G: 0x1E, B: 0x27, A: 0xFF}
	colForeground = color.NRGBA{R: 0xE9, G: 0xE6, B: 0xED, A: 0xFF}
	colMuted      = color.NRGBA{R: 0x9A, G: 0x94, B: 0xA3, A: 0xFF}
	colPrimary    = color.NRGBA{R: 0x9B, G: 0x7B, B: 0xD4, A: 0xFF}
	colSuccess    = color.NRGBA{R: 0x6F, G: 0xC2, B: 0x8C, A: 0xFF}
	colError      = color.NRGBA{R: 0xE0, G: 0x6C, B: 0x75, A: 0xFF}
)

func (darkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colBackground
	case theme.ColorNameInputBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return colSurface
	case theme.ColorNameForeground:
		return colForeground
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return colMuted
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return colPrimary
	case theme.ColorNameSuccess:
		return colSuccess
	case theme.ColorNameError:
		return colError
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (darkTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(s)
}

func (darkTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (darkTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}
