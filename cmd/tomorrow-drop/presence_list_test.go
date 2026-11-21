package main

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/discovery"
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

// An offline paired device is shown without the check icon.
func TestOfflinePairedDeviceHasNoCheckIcon(t *testing.T) {
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
	if parts := g.rowParts[obj]; parts == nil || parts.icon.Visible() {
		t.Fatalf("an offline paired device still shows the check icon")
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

	// Not checked yet: neutral, no check icon.
	obj := g.newDeviceRow()
	g.fillDeviceRow(obj, pairedItems(g)[0].row)
	parts := g.rowParts[obj]
	if parts.status.Importance == widget.SuccessImportance || parts.icon.Visible() {
		t.Fatalf("a device not checked yet looks online (importance %v, icon %v)", parts.status.Importance, parts.icon.Visible())
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
	if parts.status.Importance != widget.DangerImportance || parts.icon.Visible() {
		t.Fatalf("offline device is not red without the icon (importance %v, icon %v)", parts.status.Importance, parts.icon.Visible())
	}

	g.rememberAddress(up, peers[0].Key)
	g.checkPresence()
	g.refreshDevices()
	rows = pairedItems(g)
	g.fillDeviceRow(obj, rows[0].row)
	if parts.status.Importance != widget.SuccessImportance || !parts.icon.Visible() {
		t.Fatalf("online device does not show the online colours (importance %v, icon %v)", parts.status.Importance, parts.icon.Visible())
	}
}
