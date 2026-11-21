package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// newLabel is a label that wraps its text, so a long text never widens the
// screen beyond the window.
func newLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	return l
}

// newLabelStyled is newLabel with alignment and style.
func newLabelStyled(text string, align fyne.TextAlign, style fyne.TextStyle) *widget.Label {
	l := widget.NewLabelWithStyle(text, align, style)
	l.Wrapping = fyne.TextWrapWord
	return l
}
