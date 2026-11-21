package main

import (
	"testing"
	"time"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/ui"
)

// After renaming a device that is paired and contacted, the home list shows the
// new name at once, when the home screen is shown again.
func TestRenameShowsInTheHomeListAtOnce(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	g.openDevice(ui.DeviceRow{Name: "x", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))
	g.bg.Wait()
	now := time.Now()
	_ = g.reg.Observe(discovery.Announcement{Name: "MacBook-Air", Addr: ep, Seen: now}, now)

	peer, ok := g.pairedAt(ep)
	if !ok {
		t.Fatalf("paired device not remembered")
	}
	if err := g.renameDevice(peer, "Sala"); err != nil {
		t.Fatal(err)
	}
	g.showHome()
	if len(g.rows) != 1 || g.rows[0].Name != "Sala" {
		t.Fatalf("home rows = %+v, want the new name at once", g.rows)
	}
}

// Regression: a paired device that was discovered but not contacted since the
// application started was listed under the name it announces, and a rename did
// not reach it. The list must use the paired name for any device that is paired,
// as soon as its address is known to be the paired one.
func TestRenameReachesADeviceSeenByDiscoveryOnly(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	peer, _ := g.store.List()
	// An earlier contact, from a previous run of the application, left the address.
	g.rememberAddress(ep, peer[0].Key)
	if err := g.renameDevice(peer[0], "Sala"); err != nil {
		t.Fatal(err)
	}
	// Discovery answers from the same address, under the name it announces.
	now := time.Now()
	_ = g.reg.Observe(discovery.Announcement{Name: "MacBook-Air", Addr: ep, Seen: now}, now)
	g.showHome()
	if len(g.rows) != 1 || g.rows[0].Name != "Sala" {
		t.Fatalf("home rows = %+v, want the paired name for a device discovery finds again", g.rows)
	}
}
