package main

import (
	"time"

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

// plural picks the singular (key_one) or the plural (key_other, with the
// number) of a counted noun, so the text agrees with the number.
func (g *gui) plural(key string, n int) string {
	return g.tr.Count(key, n)
}

// countFiles reads "1 file", "2 files".
func (g *gui) countFiles(n int) string { return g.plural("count.file", n) }

// countItems reads "1 item", "2 items", for offers that may carry text or links.
func (g *gui) countItems(n int) string { return g.plural("count.item", n) }

// savedText says what was saved and, on desktop, where. On a phone each file is
// saved where the person chose, so no folder is named.
func (g *gui) savedText(n int) string {
	if g.mobile {
		return g.tr.T("gui.receive.saved_phone", g.countFiles(n))
	}
	return g.tr.T("gui.receive.saved", g.countFiles(n))
}

// displayName is how a device is called on screen: the name the person gave it
// when it is paired, the name it announces otherwise, and its address as a last
// resort.
func (g *gui) displayName(addr string) string {
	ep, err := manualEndpoint(addr)
	if err != nil {
		return addr
	}
	if p, ok := g.pairedAt(ep); ok {
		return p.Name
	}
	for _, d := range g.reg.List(time.Now()) {
		if d.Addr == ep {
			return d.Name
		}
	}
	return addr
}

// hostOf is an address without its port when the port is the usual one, which is
// not something a person needs to read; another port is kept.
func hostOf(addr string) string {
	ep, err := manualEndpoint(addr)
	if err != nil || int(ep.Port()) != defaultTLSPort {
		return addr
	}
	return ep.Addr().String()
}

// newNameLabel is a single line for a name (a device or a file): a long name is
// cut with an ellipsis instead of wrapping, so the line keeps its height. It is
// the kind of text that can be laid out compactly (gui.compact).
func newNameLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapOff
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

// boldName is a name in bold, cut with an ellipsis when it is too long.
func boldName(text string) *widget.Label {
	l := newNameLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}
