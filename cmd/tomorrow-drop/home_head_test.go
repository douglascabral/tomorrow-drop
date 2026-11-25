package main

import (
	"net/netip"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// The top of the home screen, above the device list, is kept short so the list
// gets the vertical room: the name, the address and the fingerprint fit in two
// lines of controls.
func TestHomeHeadLeavesRoomForTheList(t *testing.T) {
	// The logo is stacked above the text, so the top needs a little room.
	for name, mobile := range map[string]bool{"desktop": false, "phone": true} {
		maxHeadHeight := float32(170)
		if mobile {
			maxHeadHeight = 185 // larger text and touch spacing
		}
		g := newTestGUI(t)
		g.mobile = mobile
		g.app.Settings().SetTheme(themeFor(mobile))
		g.localAddrs = g.addrs()
		head := g.homeHead(func() {})
		if h := head.MinSize().Height; h > maxHeadHeight {
			t.Errorf("%s: the top of the home screen is %v high, want at most %v", name, h, maxHeadHeight)
		}
	}
}

// The logo sits above the name and the slogan, all centred on the same line, on
// desktop and on phones, so the top of the screen looks centred.
func TestHomeBrandIsStackedAndCentred(t *testing.T) {
	for name, mobile := range map[string]bool{"desktop": false, "phone": true} {
		t.Run(name, func(t *testing.T) { checkBrandStackedAndCentred(t, mobile) })
	}
}

func checkBrandStackedAndCentred(t *testing.T, mobile bool) {
	g := newTestGUI(t)
	g.mobile = mobile
	g.app.Settings().SetTheme(themeFor(mobile))
	g.localAddrs = g.addrs()
	head := g.homeHead(func() {})
	const width = 800
	head.Resize(fyne.NewSize(width, head.MinSize().Height))

	var logo *canvas.Image
	var name, slogan *widget.Label
	walkObjects(head, func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *canvas.Image:
			logo = v
		case *widget.Label:
			switch v.Text {
			case g.tr.T("app.nome"):
				name = v
			case g.tr.T("app.slogan"):
				slogan = v
			}
		}
	})
	if logo == nil || name == nil || slogan == nil {
		t.Fatalf("logo=%v name=%v slogan=%v not found in the top of the home screen", logo, name, slogan)
	}
	logoPos, namePos, sloganPos := absolutePosition(head, logo), absolutePosition(head, name), absolutePosition(head, slogan)
	if logoPos.Y+logo.Size().Height > namePos.Y+1 || namePos.Y > sloganPos.Y {
		t.Errorf("want the logo above the name above the slogan, got y %v, %v, %v", logoPos.Y, namePos.Y, sloganPos.Y)
	}
	for what, mid := range map[string]float32{
		"logo":   logoPos.X + logo.Size().Width/2,
		"name":   namePos.X + name.Size().Width/2,
		"slogan": sloganPos.X + slogan.Size().Width/2,
	} {
		if mid < width/2-1 || mid > width/2+1 {
			t.Errorf("the %s is centred at %v, want %v", what, mid, float32(width)/2)
		}
	}
}

// walkObjects calls fn for obj and every object below it.
func walkObjects(obj fyne.CanvasObject, fn func(fyne.CanvasObject)) {
	fn(obj)
	switch c := obj.(type) {
	case *fyne.Container:
		for _, o := range c.Objects {
			walkObjects(o, fn)
		}
	case *container.ThemeOverride:
		walkObjects(c.Content, fn)
	}
}

// absolutePosition is the position of target relative to root.
func absolutePosition(root, target fyne.CanvasObject) fyne.Position {
	var found fyne.Position
	var visit func(o fyne.CanvasObject, origin fyne.Position) bool
	visit = func(o fyne.CanvasObject, origin fyne.Position) bool {
		pos := origin.Add(o.Position())
		if o == target {
			found = pos
			return true
		}
		switch c := o.(type) {
		case *fyne.Container:
			for _, child := range c.Objects {
				if visit(child, pos) {
					return true
				}
			}
		case *container.ThemeOverride:
			return visit(c.Content, pos)
		}
		return false
	}
	visit(root, fyne.NewPos(-root.Position().X, -root.Position().Y))
	return found
}

// On desktop the address of this device is at the left corner and the
// fingerprint button at the right one, apart and never overlapping.
func TestDesktopHomeAddressAndFingerprintAreInOppositeCorners(t *testing.T) {
	g := newTestGUI(t)
	g.app.Settings().SetTheme(themeFor(false))
	g.localAddrs = g.addrs()
	head := g.homeHead(func() {})
	const width = 800
	head.Resize(fyne.NewSize(width, head.MinSize().Height))

	var address *widget.Label
	var button *widget.Button
	walkObjects(head, func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Label:
			if strings.HasPrefix(v.Text, strings.SplitN(g.tr.T("gui.home.this_device", "%"), "%", 2)[0]) {
				address = v
			}
		case *widget.Button:
			if v.Text == g.tr.T("gui.fingerprint.view") {
				button = v
			}
		}
	})
	if address == nil || button == nil {
		t.Fatalf("address=%v button=%v not found", address, button)
	}
	addrPos, btnPos := absolutePosition(head, address), absolutePosition(head, button)
	if addrPos.X > 20 {
		t.Errorf("the address starts at %v, want the left corner", addrPos.X)
	}
	if right := btnPos.X + button.Size().Width; right < width-20 {
		t.Errorf("the fingerprint button ends at %v, want the right corner (%v)", right, width)
	}
	if addrPos.X+address.Size().Width > btnPos.X {
		t.Errorf("the address (to %v) overlaps the fingerprint button (from %v)", addrPos.X+address.Size().Width, btnPos.X)
	}
}

// Regression: the address label wrapped by word, so its minimum width was tiny.
// When the window was made narrower the label was squeezed to a few units and
// its text ran over the fingerprint button. The window must stop shrinking where
// the whole address still fits beside the button.
func TestDesktopHomeAddressNeverOverlapsTheFingerprintWhenNarrow(t *testing.T) {
	g := newTestGUI(t)
	g.app.Settings().SetTheme(themeFor(false))
	g.localAddrs = []netip.Addr{netip.MustParseAddr("192.168.100.200"), netip.MustParseAddr("10.20.30.40")}
	head := g.homeHead(func() {})
	head.Resize(fyne.NewSize(head.MinSize().Width, 400))

	var address *widget.Label
	var button *widget.Button
	walkObjects(head, func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Label:
			if strings.Contains(v.Text, "192.168.100.200") {
				address = v
			}
		case *widget.Button:
			if v.Text == g.tr.T("gui.fingerprint.view") {
				button = v
			}
		}
	})
	if address == nil || button == nil {
		t.Fatalf("address=%v button=%v not found", address, button)
	}
	whole := widget.NewLabelWithStyle(address.Text, fyne.TextAlignLeading, address.TextStyle).MinSize().Width
	if address.Size().Width < whole {
		t.Errorf("at the narrowest window the address gets %v, its text needs %v", address.Size().Width, whole)
	}
	if end := absolutePosition(head, address).X + address.Size().Width; end > absolutePosition(head, button).X {
		t.Errorf("the address ends at %v, over the button that starts at %v", end, absolutePosition(head, button).X)
	}
}
