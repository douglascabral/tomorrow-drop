package main

import (
	"image/color"
	"net"
	"net/netip"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/ui"
)

// A paired device whose address accepts connections is online; once nothing
// answers there it is offline. The check reads the addresses the paired devices
// last answered at.
func TestPairedDeviceIsOnlineThenOffline(t *testing.T) {
	g := newTestGUI(t)
	up := pairedAndRemembered(t, g)

	g.checkPresence()
	if got := g.presence.State(up, time.Now()); got != discovery.PresenceOnline {
		t.Fatalf("paired device that accepts connections = %v, want online", got)
	}

	peers, _ := g.store.List()
	down := closedEndpoint(t)
	g.rememberAddress(down, peers[0].Key)
	g.checkPresence()
	if got := g.presence.State(down, time.Now()); got != discovery.PresenceOffline {
		t.Fatalf("paired device at a closed address = %v, want offline", got)
	}
}

// Every periodic query also checks the paired devices (the 30-second timer and
// Refresh both go through queryAll).
func TestQueryAllAlsoChecksPairedDevices(t *testing.T) {
	g := newTestGUI(t)
	up := pairedAndRemembered(t, g)

	if err := g.queryAll(time.Now()); err != nil {
		t.Fatalf("queryAll: %v", err)
	}
	g.bg.Wait()
	if got := g.presence.State(up, time.Now()); got != discovery.PresenceOnline {
		t.Fatalf("after queryAll the paired device is %v, want online", got)
	}
}

// Checks never overlap: while one runs, a second request is refused.
func TestPresenceChecksDoNotOverlap(t *testing.T) {
	g := newTestGUI(t)
	g.presenceBusy.Store(true)
	if g.startPresenceCheck() {
		t.Fatalf("a check started while another one is running")
	}
	g.presenceBusy.Store(false)
	_ = pairedAndRemembered(t, g)
	if !g.startPresenceCheck() {
		t.Fatalf("a check was refused when none was running")
	}
	g.bg.Wait()
}

// Unpairing forgets the reachability of the device.
func TestUnpairForgetsPresence(t *testing.T) {
	g := newTestGUI(t)
	up := pairedAndRemembered(t, g)
	g.checkPresence()

	peers, _ := g.store.List()
	if len(peers) != 1 {
		t.Fatalf("paired devices = %d, want 1", len(peers))
	}
	if err := g.unpairDevice(peers[0]); err != nil {
		t.Fatalf("unpairDevice: %v", err)
	}
	if got := g.presence.State(up, time.Now()); got != discovery.PresenceUnknown {
		t.Fatalf("presence after unpair = %v, want unknown", got)
	}
}

// Online paired devices come first in the home list; each row says its state.
func TestOnlinePairedDevicesAreListedFirst(t *testing.T) {
	g := newTestGUI(t)
	up := pairedAndRemembered(t, g)
	_ = pairedDevice(t, g) // second paired device, no address known yet
	peers, _ := g.store.List()
	if len(peers) != 2 {
		t.Fatalf("paired devices = %d, want 2", len(peers))
	}
	g.checkPresence()
	g.refreshDevices()

	rows := pairedItems(g)
	if len(rows) != 2 {
		t.Fatalf("paired rows = %d, want 2", len(rows))
	}
	if rows[0].row.Address != up.String() || rows[0].row.Presence != discovery.PresenceOnline {
		t.Fatalf("first paired row = %+v, want the online device", rows[0].row)
	}
	if rows[0].row.StatusKey != "ui.status.online" {
		t.Fatalf("online row status key = %q", rows[0].row.StatusKey)
	}
}

// An offline paired device shows the red cross, never the online check.
func TestOfflinePairedDeviceShowsTheRedCross(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	peers, _ := g.store.List()
	g.rememberAddress(closedEndpoint(t), peers[0].Key)
	_ = ep
	g.checkPresence()
	g.refreshDevices()

	rows := pairedItems(g)
	if len(rows) != 1 || rows[0].row.StatusKey != "ui.status.offline" {
		t.Fatalf("offline paired row = %+v", rows)
	}
	obj := g.newDeviceRow()
	g.fillDeviceRow(obj, rows[0].row)
	if parts := g.rowParts[obj]; parts == nil || parts.icon.Resource == nil || parts.icon.Resource.Name() != crossIconName() {
		t.Fatalf("an offline paired device does not show the red cross")
	}
}

// closedEndpoint returns an address where nothing listens.
func closedEndpoint(t *testing.T) netip.AddrPort {
	t.Helper()
	ln, ep := listenOnLoopback(t)
	_ = ln.Close()
	return ep
}

// listenOnLoopback opens a TCP listener on loopback.
func listenOnLoopback(t *testing.T) (net.Listener, netip.AddrPort) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln, netip.MustParseAddrPort(ln.Addr().String())
}

// Regression: an offline paired device was shown in the colour of an online one
// when it had no current check result, and the check icon came with it. Online
// is green with the check icon; offline is red without it; a device not checked
// yet is neutral, never green.
func TestPresenceColoursMatchTheState(t *testing.T) {
	g := newTestGUI(t)
	up := pairedAndRemembered(t, g)
	g.refreshDevices()

	// Not checked yet is not confirmed online, so it reads as offline: red, with
	// no grey state and no check icon.
	obj := g.newDeviceRow()
	unchecked := pairedItems(g)[0].row
	if unchecked.StatusKey != "ui.status.offline" {
		t.Fatalf("a device not checked yet has status %q, want offline", unchecked.StatusKey)
	}
	g.fillDeviceRow(obj, unchecked)
	parts := g.rowParts[obj]
	if parts.status.Importance != widget.DangerImportance || parts.icon.Resource.Name() != crossIconName() {
		t.Fatalf("a device not checked yet is not red with the cross (importance %v)", parts.status.Importance)
	}

	peers, _ := g.store.List()
	g.rememberAddress(closedEndpoint(t), peers[0].Key)
	g.checkPresence()
	g.refreshDevices()

	rows := pairedItems(g)
	if len(rows) != 1 || rows[0].row.Address == up.String() {
		t.Fatalf("setup: paired rows = %+v", rows)
	}
	g.fillDeviceRow(obj, rows[0].row)
	if parts.status.Importance != widget.DangerImportance || parts.icon.Resource.Name() != crossIconName() {
		t.Fatalf("offline device is not red with the cross (importance %v)", parts.status.Importance)
	}

	g.rememberAddress(up, peers[0].Key)
	g.checkPresence()
	g.refreshDevices()
	rows = pairedItems(g)
	g.fillDeviceRow(obj, rows[0].row)
	if parts.status.Importance != widget.SuccessImportance || parts.icon.Resource.Name() != checkIconName() {
		t.Fatalf("online device does not show the green check (importance %v)", parts.status.Importance)
	}
}

// checkIconName and crossIconName name the icons of an online and an offline
// device, so a test can tell them apart.
func checkIconName() string { return theme.NewSuccessThemedResource(theme.ConfirmIcon()).Name() }
func crossIconName() string { return theme.NewErrorThemedResource(theme.CancelIcon()).Name() }

// renderedTextColour returns the colour the first piece of text inside w is
// drawn with, which is what the person sees (not just the Importance field).
func renderedTextColour(w fyne.Widget) color.Color {
	for _, obj := range test.WidgetRenderer(w).Objects() {
		if txt, ok := obj.(*canvas.Text); ok {
			return txt.Color
		}
		if inner, ok := obj.(fyne.Widget); ok {
			if c := renderedTextColour(inner); c != nil {
				return c
			}
		}
	}
	return nil
}

func sameColour(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// Regression: the list reuses a row for other devices, and the status kept the
// colour of the device the row showed before: Online was drawn red and Offline
// green. The colour drawn must follow the device, however the row is reused.
func TestStatusColourFollowsTheDeviceWhenARowIsReused(t *testing.T) {
	g := newTestGUI(t)
	online := ui.DeviceRow{Name: "ipad", Address: "10.0.0.2:47521", Trusted: true, StatusKey: "ui.status.online", Presence: discovery.PresenceOnline}
	offline := ui.DeviceRow{Name: "moto", Address: "10.0.0.3:47521", Trusted: true, StatusKey: "ui.status.offline", Presence: discovery.PresenceOffline}
	green := theme.Color(theme.ColorNameSuccess)
	red := theme.Color(theme.ColorNameError)

	obj := g.newDeviceRow()
	parts := g.rowParts[obj]
	for i, step := range []struct {
		row  ui.DeviceRow
		want color.Color
		name string
	}{{online, green, "online"}, {offline, red, "offline"}, {online, green, "online again"}, {offline, red, "offline again"}} {
		g.fillDeviceRow(obj, step.row)
		if got := renderedTextColour(parts.status); got == nil || !sameColour(got, step.want) {
			t.Fatalf("step %d (%s): the status is drawn in %v, want %v", i, step.name, got, step.want)
		}
	}
}

// Every device line keeps its icon column, so names line up: online has a green
// check, offline a red cross, and a found device a neutral computer.
func TestEveryDeviceLineKeepsItsIconColumn(t *testing.T) {
	g := newTestGUI(t)
	rows := []ui.DeviceRow{
		{Name: "a", Address: "10.0.0.2:47521", Trusted: true, StatusKey: "ui.status.online", Presence: discovery.PresenceOnline},
		{Name: "b", Address: "10.0.0.3:47521", Trusted: true, StatusKey: "ui.status.offline"},
		{Name: "c", Address: "10.0.0.4:47521", StatusKey: "ui.status.discovered"},
	}
	for _, row := range rows {
		obj := g.newDeviceRow()
		g.fillDeviceRow(obj, row)
		if parts := g.rowParts[obj]; parts == nil || !parts.icon.Visible() || parts.icon.Resource == nil {
			t.Fatalf("the line of %q has no icon, so its name moves out of line", row.Name)
		}
	}
}
