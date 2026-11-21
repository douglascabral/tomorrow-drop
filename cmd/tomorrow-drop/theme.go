package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// darkTheme is the dark reference theme (PRD §19.2): neutral colors with
// discreet purple accents and success or error indicators.
type darkTheme struct{}

var (
	colBackground = color.NRGBA{R: 0x14, G: 0x13, B: 0x1C, A: 0xFF}
	colSurface    = color.NRGBA{R: 0x1E, G: 0x1C, B: 0x27, A: 0xFF}
	colForeground = color.NRGBA{R: 0xE9, G: 0xE6, B: 0xED, A: 0xFF}
	colMuted      = color.NRGBA{R: 0x9A, G: 0x94, B: 0xA3, A: 0xFF}
	colPrimary    = color.NRGBA{R: 0x6E, G: 0x4B, B: 0xFF, A: 0xFF}
	colSuccess    = color.NRGBA{R: 0x3D, G: 0xD6, B: 0x8C, A: 0xFF}
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

// touchTheme enlarges the spacing, text and icons of the dark theme for phones
// and tablets, so buttons and list rows are easy to tap.
type touchTheme struct {
	darkTheme
}

func (touchTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameInnerPadding:
		return 20
	case theme.SizeNamePadding:
		return 12
	case theme.SizeNameText:
		return 18
	case theme.SizeNameInlineIcon:
		return 28
	}
	return darkTheme{}.Size(n)
}

// themeFor returns the theme of the application: touch-sized on phones and
// tablets, compact on desktop.
func themeFor(mobile bool) fyne.Theme {
	if mobile {
		return touchTheme{}
	}
	return darkTheme{}
}

// touchMinHeight is the smallest height of a control a person taps on a phone (dp).
const touchMinHeight = 64

// touch keeps a control at least touchMinHeight high on phones and tablets. The
// theme does not reach every control, so the height is given by the layout.
// On desktop the control is returned unchanged.
func (g *gui) touch(obj fyne.CanvasObject) fyne.CanvasObject {
	if !g.mobile {
		return obj
	}
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, touchMinHeight))
	return container.NewStack(spacer, obj)
}

// compactTheme keeps the text size of the touch theme but not its padding, for
// blocks of text. The padding that makes a button easy to tap would make every
// line of text as high as a button.
type compactTheme struct {
	darkTheme
}

func (compactTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameText:
		return 18
	}
	return darkTheme{}.Size(n)
}

// compact lays out a block of text compactly on a phone. Controls are not
// passed through it, so they keep the large touch size. On desktop the object
// is returned as it is.
func (g *gui) compact(obj fyne.CanvasObject) fyne.CanvasObject {
	if !g.mobile {
		return obj
	}
	return container.NewThemeOverride(obj, compactTheme{})
}
