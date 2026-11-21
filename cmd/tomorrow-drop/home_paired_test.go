package main

import (
	"net/netip"
	"testing"
	"time"

	"tomorrow-drop/internal/discovery"
)

// pairedItems returns the device lines of the home list that belong to paired
// devices, and foundItems those of devices only discovery has seen.
func pairedItems(g *gui) []homeItem {
	var out []homeItem
	for _, it := range g.homeList {
		if it.kind == homeDevice && it.row.Trusted {
			out = append(out, it)
		}
	}
	return out
}

func foundItems(g *gui) []homeItem {
	var out []homeItem
	for _, it := range g.homeList {
		if it.kind == homeDevice && !it.row.Trusted {
			out = append(out, it)
		}
	}
	return out
}

// pairedAndRemembered pairs a device and remembers the address it answered at,
// as a connection does, so that the home screen can list it by that address.
func pairedAndRemembered(t *testing.T, g *gui) netip.AddrPort {
	t.Helper()
	ep := pairedDevice(t, g)
	peers, err := g.store.List()
	if err != nil || len(peers) != 1 {
		t.Fatalf("paired devices = %v (%v), want one", peers, err)
	}
	g.rememberAddress(ep, peers[0].Key)
	return ep
}

// A paired device stays on the home screen, in its own list, even when no
// announcement reaches this device (another VLAN, or the app was reopened).
func TestHomeListsPairedDevicesInTheirOwnSection(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedAndRemembered(t, g)
	g.refreshDevices()

	paired := pairedItems(g)
	if len(paired) != 1 || paired[0].row.Name != "other" || paired[0].row.Address != ep.String() {
		t.Fatalf("paired section = %+v, want the paired device at %s", paired, ep)
	}
	if len(foundItems(g)) != 0 {
		t.Fatalf("the paired device is listed again among the found devices")
	}
	if !homeHasHeader(g, g.tr.T("gui.home.trusted")) || !homeHasHeader(g, g.tr.T("gui.home.devices")) {
		t.Fatalf("the home list does not title its two sections")
	}
}

// A device that discovery sees at the address of a paired device is the same
// device: it is listed once, as paired.
func TestPairedDeviceIsNotRepeatedAmongFoundDevices(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedAndRemembered(t, g)
	if err := g.reg.Observe(discovery.Announcement{Name: "MacBook", Addr: ep, Seen: time.Now()}, time.Now()); err != nil {
		t.Fatal(err)
	}
	g.refreshDevices()
	if n := len(pairedItems(g)); n != 1 {
		t.Fatalf("paired device listed %d times, want 1", n)
	}
	if n := len(foundItems(g)); n != 0 {
		t.Fatalf("paired device listed among found devices: %d", n)
	}
}

// A paired device whose address is not known yet cannot be opened from the
// home screen, and says so.
func TestPairedDeviceWithoutAddressSaysItCannotBeOpened(t *testing.T) {
	g := newTestGUI(t)
	_ = pairedDevice(t, g)
	g.refreshDevices()
	paired := pairedItems(g)
	if len(paired) != 1 || paired[0].row.Address != "" || paired[0].row.StatusKey != "ui.status.no_address" {
		t.Fatalf("paired device without address = %+v", paired)
	}
}

// Tapping a paired device opens its actions (send, receive) at once.
func TestTappingAPairedDeviceOpensItsActions(t *testing.T) {
	g := newTestGUI(t)
	_ = pairedAndRemembered(t, g)
	g.showHome()
	index := -1
	for i, it := range g.homeList {
		if it.kind == homeDevice && it.row.Trusted {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("no paired device in the home list")
	}
	g.openHomeItem(index)
	waitForButton(t, g, g.tr.T("gui.device.send_files"))
}

func homeHasHeader(g *gui, title string) bool {
	for _, it := range g.homeList {
		if it.kind == homeHeader && it.title == title {
			return true
		}
	}
	return false
}
