package main

import (
	"testing"
	"time"

	"tomorrow-drop/internal/ui"
)

// Regression: a device connected to by IP address vanished from the list as
// soon as the connection ended. Only announced devices were in the list, and a
// device on another VLAN is never announced here, so trust could not be marked
// and the person could not act on it again. Such a device stays listed while
// the application runs, even when nothing answers from it.
func TestDeviceConnectedByAddressStaysListedDuringTheSession(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)

	g.connectTo(ui.DeviceRow{Name: ep.String(), Address: ep.String(), StatusKey: "ui.status.discovered"}, true)
	g.bg.Wait()

	g.refreshDevices()
	if len(g.rows) != 1 || !g.rows[0].Trusted || g.rows[0].Name != "other" {
		t.Fatalf("rows = %+v, want the device listed as paired", g.rows)
	}
	// The announcement lifetime is 90 seconds; the device must outlive it.
	if got := len(g.reg.List(time.Now().Add(5 * time.Minute))); got != 1 {
		t.Fatalf("device connected by address expired from the list: %d devices", got)
	}
}
