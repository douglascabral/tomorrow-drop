package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// The received text can be copied to the clipboard of the device.
func TestReceivedTextCanBeCopiedToTheClipboard(t *testing.T) {
	g := newTestGUI(t)
	g.showText("olá, mundo")

	button := findButton(g.win.Content(), g.tr.T("gui.receive.copy"))
	if button == nil {
		t.Fatalf("received text has no copy button; buttons = %v", buttonLabels(g.win.Content()))
	}
	button.OnTapped()
	if got := g.app.Clipboard().Content(); got != "olá, mundo" {
		t.Fatalf("clipboard = %q, want the received text", got)
	}
}

// findButton returns the button with the text under obj, or nil.
func findButton(obj fyne.CanvasObject, text string) *widget.Button {
	switch o := obj.(type) {
	case *widget.Button:
		if o.Text == text {
			return o
		}
	case *container.Scroll:
		return findButton(o.Content, text)
	case *container.ThemeOverride:
		return findButton(o.Content, text)
	case *fyne.Container:
		for _, c := range o.Objects {
			if b := findButton(c, text); b != nil {
				return b
			}
		}
	}
	return nil
}
