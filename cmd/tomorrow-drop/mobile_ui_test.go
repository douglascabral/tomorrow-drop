package main

import (
	"net/netip"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

// The device list must not show the device the application runs on.
func TestDiscoveryIgnoresThisDevice(t *testing.T) {
	g := newTestGUI(t)
	g.localAddrs = []netip.Addr{netip.MustParseAddr("10.0.0.5")}
	now := time.Now()
	g.observe(discovery.Announcement{Name: "this-ipad", Addr: netip.AddrPortFrom(g.localAddrs[0], defaultTLSPort), Seen: now})
	g.observe(discovery.Announcement{Name: "phone", Addr: netip.MustParseAddrPort("10.0.0.6:47521"), Seen: now})

	devices := g.reg.List(now)
	if len(devices) != 1 || devices[0].Name != "phone" {
		t.Fatalf("discovered devices = %+v, want only the phone", devices)
	}
}

// Touch targets are large on phones and tablets: buttons at least 64 dp high and
// device rows at least 72 dp, well above the 48 dp guideline (docs/MOBILE.md, task 060).
func TestMobileTouchTargetsAreLarge(t *testing.T) {
	g := newTestGUI(t)
	g.app.Settings().SetTheme(themeFor(true))

	button := widget.NewButton("Enviar arquivos", nil)
	if h := button.MinSize().Height; h < 64 {
		t.Fatalf("button height = %v, want at least 64", h)
	}
	row := g.newDeviceRow()
	if h := row.MinSize().Height; h < 72 {
		t.Fatalf("device row height = %v, want at least 72", h)
	}
}

// Desktop keeps its compact sizes; only mobile gets the larger targets.
func TestDesktopKeepsCompactTheme(t *testing.T) {
	if got, want := themeFor(false).Size(theme.SizeNameInnerPadding), (darkTheme{}).Size(theme.SizeNameInnerPadding); got != want {
		t.Fatalf("desktop inner padding = %v, want %v", got, want)
	}
}

// A paired device is marked with an icon; a discovered one is not.
func TestPairedDeviceRowShowsIcon(t *testing.T) {
	g := newTestGUI(t)
	// The check icon means online: a paired device that answered its last check.
	// A paired device not checked yet is neutral, so it has no icon.
	paired := g.newDeviceRow()
	g.fillDeviceRow(paired, ui.DeviceRow{Name: "laptop", Address: "10.0.0.2:47521", Trusted: true, StatusKey: "ui.status.online", Presence: discovery.PresenceOnline})
	if parts := g.rowParts[paired]; parts == nil || !parts.icon.Visible() || parts.icon.Resource == nil {
		t.Fatalf("online paired device row has no icon")
	}
	unchecked := g.newDeviceRow()
	g.fillDeviceRow(unchecked, ui.DeviceRow{Name: "laptop", Address: "10.0.0.2:47521", Trusted: true, StatusKey: "ui.status.offline"})
	if parts := g.rowParts[unchecked]; parts == nil || parts.icon.Resource == nil || parts.icon.Resource.Name() == checkIconName() {
		t.Fatalf("paired device not checked yet shows the online icon")
	}

	found := g.newDeviceRow()
	g.fillDeviceRow(found, ui.DeviceRow{Name: "phone", Address: "10.0.0.3:47521", StatusKey: "ui.status.discovered"})
	if parts := g.rowParts[found]; parts == nil || parts.icon.Resource == nil || parts.icon.Resource.Name() == checkIconName() || parts.icon.Resource.Name() == crossIconName() {
		t.Fatalf("discovered device row shows a paired icon")
	}
}

// buttonLabels collects the text of every button under obj.
func buttonLabels(obj fyne.CanvasObject) []string {
	switch o := obj.(type) {
	case *widget.Button:
		return []string{o.Text}
	case *container.Scroll:
		return buttonLabels(o.Content)
	case *container.ThemeOverride:
		return buttonLabels(o.Content)
	case *fyne.Container:
		var out []string
		for _, child := range o.Objects {
			out = append(out, buttonLabels(child)...)
		}
		return out
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// The home screen offers Send and Receive only once a device is chosen from
// the list; the device screen carries them.
func TestHomeHidesSendAndReceiveUntilDeviceIsChosen(t *testing.T) {
	g := newTestGUI(t)
	g.showHome()
	labels := buttonLabels(g.win.Content())
	for _, key := range []string{"gui.home.send", "gui.home.receive"} {
		if contains(labels, g.tr.T(key)) {
			t.Errorf("home shows %q before a device is chosen; buttons = %v", key, labels)
		}
	}

	row := ui.DeviceRow{Name: "laptop", Address: "10.0.0.2:47521", Trusted: true, StatusKey: "ui.status.trusted"}
	g.showTrustedDevice(row)
	labels = buttonLabels(g.win.Content())
	for _, key := range []string{"gui.device.send_files", "gui.home.receive"} {
		if !contains(labels, g.tr.T(key)) {
			t.Errorf("device screen lacks %q; buttons = %v", key, labels)
		}
	}
}

// labelTexts collects the text of every label and button under obj.
func labelTexts(obj fyne.CanvasObject) []string {
	switch o := obj.(type) {
	case *widget.Label:
		return []string{o.Text}
	case *widget.Button:
		return []string{o.Text}
	case *container.Scroll:
		return labelTexts(o.Content)
	case *container.ThemeOverride:
		return labelTexts(o.Content)
	case *fyne.Container:
		var out []string
		for _, child := range o.Objects {
			out = append(out, labelTexts(child)...)
		}
		return out
	}
	return nil
}

// The home screen shows this device's current address, so two open devices can
// be told apart without looking up their numbers.
func TestHomeShowsThisDeviceAddress(t *testing.T) {
	g := newTestGUI(t)
	g.addrs = func() []netip.Addr { return []netip.Addr{netip.MustParseAddr("192.168.1.42")} }
	g.showHome()
	want := g.tr.T("gui.home.this_device", "192.168.1.42")
	if !contains(labelTexts(g.win.Content()), want) {
		t.Fatalf("home does not show %q; texts = %v", want, labelTexts(g.win.Content()))
	}
	if g.localAddrs == nil || g.localAddrs[0].String() != "192.168.1.42" {
		t.Fatalf("own address for the self filter = %v, want the current one", g.localAddrs)
	}
}

// The name this device announces must follow the discovery rules on every
// platform, so the other devices always accept it (docs/DISCOVERY.md).
func TestAnnouncedNameFollowsDiscoveryRules(t *testing.T) {
	name := announcedName()
	if err := trust.ValidateName(name); err != nil {
		t.Fatalf("announced name %q is not valid: %v", name, err)
	}
}

// Regression: going to the background must close the incoming listener before
// it returns. The close used to run in a goroutine, so a connection could still
// be accepted right after the call (TestBackgroundClosesIncomingListenerAndForegroundReopensIt
// failed about one run in twenty under -race).
func TestBackgroundClosesListenerBeforeReturning(t *testing.T) {
	g := newTestGUI(t)
	g.tlsPort = 0
	g.discPort = 0
	for i := 0; i < 50; i++ {
		g.setAvailable(true)
		port := g.boundTLSPort()
		if port == 0 {
			t.Fatalf("round %d: listener did not bind", i)
		}
		g.setAvailable(false)
		if canDial(port) {
			t.Fatalf("round %d: listener still accepts connections right after going to the background", i)
		}
	}
}

// Every control a person taps or types into is large on phones: the text
// fields and the pickers as well as the buttons, so the layout is the same on
// all screens.
func TestMobileControlsAreLargeOnEveryScreen(t *testing.T) {
	g := newTestGUI(t)
	g.app.Settings().SetTheme(themeFor(true))
	g.mobile = true

	controls := map[string]fyne.CanvasObject{
		"text field": widget.NewEntry(),
		"picker":     g.touch(widget.NewSelect([]string{"auto", "en"}, nil)),
		"button":     widget.NewButton("Enviar", nil),
	}
	for name, c := range controls {
		if h := c.MinSize().Height; h < 64 {
			t.Errorf("%s height = %v, want at least 64", name, h)
		}
	}
}

// touch keeps a control at least touchMinHeight high on phones, whatever the
// theme gives it (the picker of the settings is shorter than the other controls
// otherwise). On desktop the control is returned unchanged.
func TestTouchKeepsPickersLargeOnlyOnMobile(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = true
	if h := g.touch(widget.NewSelect([]string{"auto", "en"}, nil)).MinSize().Height; h < touchMinHeight {
		t.Fatalf("picker height on mobile = %v, want at least %v", h, touchMinHeight)
	}
	g.mobile = false
	picker := widget.NewSelect([]string{"auto", "en"}, nil)
	if g.touch(picker) != fyne.CanvasObject(picker) {
		t.Fatalf("desktop control was wrapped; it must stay as it is")
	}
}

// Other devices find this one only while the app is open and in front, on every
// platform (desktop loses the front when its window is not focused). The home
// screen says so, in a small line, so the person knows before leaving the app.
func TestHomeSaysItIsFoundOnlyWhileOpen(t *testing.T) {
	g := newTestGUI(t)
	g.showHome()
	want := g.tr.T("gui.home.visible_only_open")
	if !contains(labelTexts(g.win.Content()), want) {
		t.Fatalf("home does not say the app is found only while open; texts = %v", labelTexts(g.win.Content()))
	}
}
