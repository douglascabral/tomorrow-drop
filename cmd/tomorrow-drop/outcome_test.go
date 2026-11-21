package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// A failed or cancelled transfer must not show the success mark.
func TestFailedTransferShowsAnErrorMark(t *testing.T) {
	g := newTestGUI(t)
	g.showOutcome(false, g.tr.T("gui.send.failed"), "x", nil)
	mark := findImage(g.win.Content())
	if mark == nil {
		t.Fatalf("result screen has no mark")
	}
	if mark.Resource == theme.ConfirmIcon() {
		t.Fatalf("a failed transfer shows the success mark")
	}
}

func TestCompletedTransferShowsTheSuccessMark(t *testing.T) {
	g := newTestGUI(t)
	g.showOutcome(true, g.tr.T("gui.send.done"), "x", nil)
	if mark := findImage(g.win.Content()); mark == nil || mark.Resource != theme.ConfirmIcon() {
		t.Fatalf("a completed transfer lacks the success mark")
	}
}

// findImage returns the first image under obj, or nil.
func findImage(obj fyne.CanvasObject) *canvas.Image {
	switch o := obj.(type) {
	case *canvas.Image:
		return o
	case *container.Scroll:
		return findImage(o.Content)
	case *fyne.Container:
		for _, c := range o.Objects {
			if img := findImage(c); img != nil {
				return img
			}
		}
	}
	return nil
}
