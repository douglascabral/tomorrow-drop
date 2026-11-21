package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// On a wide desktop window the content is a centered column of at most
// maxColumnWidth, instead of stretching across the whole window.
func TestColumnCentersContentWithAMaximumWidth(t *testing.T) {
	obj := canvas.NewRectangle(nil)
	objs := []fyne.CanvasObject{obj}
	(&maxWidthLayout{max: 560}).Layout(objs, fyne.NewSize(900, 640))
	if obj.Size().Width != 560 {
		t.Fatalf("width = %v, want 560", obj.Size().Width)
	}
	if obj.Position().X != 170 {
		t.Fatalf("x = %v, want 170 (centered)", obj.Position().X)
	}
}

func TestColumnUsesTheWholeNarrowWindow(t *testing.T) {
	obj := canvas.NewRectangle(nil)
	objs := []fyne.CanvasObject{obj}
	(&maxWidthLayout{max: 560}).Layout(objs, fyne.NewSize(420, 640))
	if obj.Size().Width != 420 || obj.Position().X != 0 {
		t.Fatalf("narrow window: width %v x %v, want 420 at 0", obj.Size().Width, obj.Position().X)
	}
}

// Screens on desktop are wrapped in the column; phones keep the full width.
func TestDesktopScreensUseTheColumnAndPhonesDoNot(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = false
	g.showHome()
	if l, ok := g.win.Content().(*fyne.Container); !ok || !isColumn(l) {
		t.Fatalf("desktop screen is not in the centered column")
	}

	g.mobile = true
	g.showHome()
	if l, ok := g.win.Content().(*fyne.Container); ok && isColumn(l) {
		t.Fatalf("phone screen must keep the full width")
	}
}

func isColumn(c *fyne.Container) bool {
	_, ok := c.Layout.(*maxWidthLayout)
	return ok
}
