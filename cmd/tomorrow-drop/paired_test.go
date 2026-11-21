package main

import (
	"testing"

	"tomorrow-drop/internal/trust"
)

// The name of a paired device can be changed by the person; the list shows it.
func TestRenamingAPairedDeviceUpdatesTheList(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	peer, ok := g.pairedAt(ep)
	if !ok {
		peers, _ := g.store.List()
		if len(peers) == 0 {
			t.Fatalf("no paired device")
		}
		peer = peers[0]
		g.rememberPaired(ep, peer.Key)
	}
	if err := g.renameDevice(peer, "Sala"); err != nil {
		t.Fatalf("renameDevice: %v", err)
	}
	peers, _ := g.store.List()
	if len(peers) != 1 || peers[0].Name != "Sala" {
		t.Fatalf("stored name = %+v, want Sala", peers)
	}
	if p, ok := g.pairedAt(ep); !ok || p.Name != "Sala" {
		t.Fatalf("remembered name = %+v, want Sala", p)
	}
}

// A device that is not paired cannot be renamed.
func TestRenamingAnUnpairedDeviceFails(t *testing.T) {
	g := newTestGUI(t)
	if err := g.renameDevice(trust.Peer{Name: "x"}, "y"); err == nil {
		t.Fatalf("renaming a device that is not paired must fail")
	}
}

// Regression: a device unpaired on one side and still paired on the other made
// the next pairing fail. When the paired side is asked to pair again, it is told
// it still has the device paired, and can unpair it so the pairing can be done.
func TestPeerAskingToPairAgainIsOfferedAnUnpair(t *testing.T) {
	g := newTestGUI(t)
	_ = pairedDevice(t, g)
	peers, err := g.store.List()
	if err != nil || len(peers) != 1 {
		t.Fatalf("paired devices = %v (%v), want one", peers, err)
	}
	if !g.pairedHere(peers[0]) {
		t.Fatalf("a device that is paired here must be offered an unpair when it asks to pair again")
	}
	if g.pairedHere(trust.Peer{Key: []byte("no-such-device-key-000000000000")}) {
		t.Fatalf("an unknown device is not paired here")
	}
}
