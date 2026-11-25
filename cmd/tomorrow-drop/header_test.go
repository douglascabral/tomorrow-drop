package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Regression: the back button took room on the left only, so the title of a
// screen (for example a device) was centred in what was left, not in the window.
func TestHeaderTitleIsCentredInTheWindow(t *testing.T) {
	const width = 600
	for name, mobile := range map[string]bool{"desktop": false, "phone": true} {
		g := newTestGUI(t)
		g.mobile = mobile
		g.app.Settings().SetTheme(themeFor(mobile))
		head := g.headerWith("Device", func() {})
		head.Resize(fyne.NewSize(width, head.MinSize().Height))
		var title *widget.Label
		walkObjects(head, func(o fyne.CanvasObject) {
			if l, ok := o.(*widget.Label); ok && l.Text == "Device" {
				title = l
			}
		})
		if title == nil {
			t.Fatalf("%s: title not found", name)
		}
		pos := absolutePosition(head, title)
		mid := pos.X + title.Size().Width/2
		if mid < width/2-1 || mid > width/2+1 {
			t.Errorf("%s: the title is centred at %v, want %v", name, mid, float32(width)/2)
		}
	}
}

func TestHeaderTitleNeverCoversTheBackButton(t *testing.T) {
	g := newTestGUI(t)
	head := g.headerWith("A very long title that has to wrap instead of running under the back button", func() {})
	head.Resize(fyne.NewSize(300, head.MinSize().Height))
	var back *widget.Button
	var title *widget.Label
	walkObjects(head, func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Button:
			back = v
		case *widget.Label:
			title = v
		}
	})
	if back == nil || title == nil {
		t.Fatal("back button or title not found")
	}
	if absolutePosition(head, title).X < back.Size().Width {
		t.Errorf("the title starts at %v, under the back button (%v wide)", absolutePosition(head, title).X, back.Size().Width)
	}
}
