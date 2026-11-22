package main

import (
	"context"
	"net"
	"net/netip"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/pairing"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

// pairedDevice starts a device that answers like another phone or computer and
// pairs g with it once, as a person does on first contact. The other device
// names g "gui"; g names it "other". It returns the other device's address.
func pairedDevice(t *testing.T, g *gui) netip.AddrPort {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	other, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	otherStore := trust.Open(filepath.Join(t.TempDir(), "trust.json"))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// The fake device writes its own trust file in a temporary folder. Stop it and
	// wait for it before the folder is removed, or the cleanup fails with
	// "directory not empty" when the device is still saving (TestPeerAskingToPairAgain...).
	done := make(chan struct{})
	go func() {
		defer close(done)
		answerLikeAPhone(ctx, ln, other, otherStore)
	}()
	t.Cleanup(func() {
		cancel()
		_ = ln.Close()
		<-done
	})
	ep := netip.MustParseAddrPort(ln.Addr().String())

	s, err := transport.Dial(ctx, ep, g.id, g.store)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := pairing.Run(ctx, s, g.id, g.store, yesPrompts("other")); err != nil {
		t.Fatalf("pairing: %v", err)
	}
	_ = s.Close()
	return ep
}

// Regression: opening a device that is already trusted must show its actions
// (send, receive), not the "already trusted" alert. The discovered row does not
// know about trust, so the device has to be contacted first.
func TestOpeningATrustedDeviceShowsItsActions(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)

	// The list row does not know the device is trusted (only an authenticated
	// connection proves it, docs/DISCOVERY.md), so Trusted is false here.
	g.openDevice(ui.DeviceRow{Name: "other", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))
}

// The paired name, chosen by the person, replaces the name the device announces
// (its computer name) once the connection has shown who the device is.
func TestPairedNameReplacesTheAnnouncedNameInTheList(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	if err := g.reg.Observe(discovery.Announcement{Name: "MacBook-Air-de-Douglas", Addr: ep, Seen: time.Now()}, time.Now()); err != nil {
		t.Fatal(err)
	}

	g.openDevice(ui.DeviceRow{Name: "MacBook-Air-de-Douglas", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))

	g.refreshDevices()
	if len(g.rows) != 1 || g.rows[0].Name != "other" || !g.rows[0].Trusted {
		t.Fatalf("rows = %+v, want the device listed as paired under its paired name", g.rows)
	}
}

// Regression: the device screen showed "—" for the pairing date and the
// fingerprint, because it looked the device up by its announced name.
func TestDeviceScreenShowsRealPairingDetails(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	g.openDevice(ui.DeviceRow{Name: "x", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))

	texts := labelTexts(g.win.Content())
	if !contains(texts, "other") {
		t.Fatalf("device screen does not show the paired name; texts = %v", texts)
	}
	for _, s := range texts {
		if s == "—" || s == g.tr.T("gui.device.paired_at", "—") {
			t.Fatalf("device screen shows a placeholder instead of the details: %q", s)
		}
	}
}

// Regression: "Remove trust" did nothing for most devices, for the same reason.
func TestUnpairRemovesTrustFromTheDevice(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	g.openDevice(ui.DeviceRow{Name: "x", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))

	peer, ok := g.pairedAt(ep)
	if !ok {
		t.Fatalf("the paired device is not remembered")
	}
	if err := g.unpairDevice(peer); err != nil {
		t.Fatalf("unpairDevice: %v", err)
	}
	if g.store.IsTrusted(peer.Key) {
		t.Fatalf("the device is still trusted after unpairing")
	}
	if _, ok := g.pairedAt(ep); ok {
		t.Fatalf("the unpaired device is still remembered as paired")
	}
}

// The list of paired devices can unpair a device even when it is not online.
func TestPairedListOffersUnpair(t *testing.T) {
	g := newTestGUI(t)
	_ = pairedDevice(t, g)
	g.showTrusted()
	if !contains(buttonLabels(g.win.Content()), g.tr.T("gui.trusted.remove")) {
		t.Fatalf("paired list has no unpair action; buttons = %v", buttonLabels(g.win.Content()))
	}
}

// answerLikeAPhone accepts connections as the other device does: trusted peers
// are served, unknown ones take part in pairing.
func answerLikeAPhone(ctx context.Context, ln net.Listener, id *identity.Identity, store *trust.Store) {
	// Connections are finished before this returns, so nothing is still saving
	// when the caller goes on.
	var conns sync.WaitGroup
	defer conns.Wait()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conns.Add(1)
		go func() {
			defer conns.Done()
			s, err := transport.Server(ctx, conn, id, store)
			if err != nil {
				return
			}
			defer func() { _ = s.Close() }()
			if s.State() != protocol.StateTrusted {
				_, _ = pairing.Run(ctx, s, id, store, yesPrompts("gui"))
				return
			}
			<-ctx.Done()
		}()
	}
}

func yesPrompts(name string) pairing.Prompts {
	return pairing.Prompts{
		Confirm: func(_, _ identity.Fingerprint) bool { return true },
		Name:    func(identity.Fingerprint) string { return name },
	}
}

// waitForButton polls the window until a button with the text appears.
func waitForButton(t *testing.T, g *gui, text string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if contains(buttonLabels(g.win.Content()), text) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("window never showed the button %q; buttons = %v", text, buttonLabels(g.win.Content()))
}

// offlinePairedDevice pairs g with a device and then moves it to an address
// where nothing answers, as a paired device that is switched off or away.
// Only the trust store and the remembered address know about it, as after a
// restart of the application.
func offlinePairedDevice(t *testing.T, g *gui, address bool) ui.DeviceRow {
	t.Helper()
	_ = pairedDevice(t, g)
	peers, err := g.store.List()
	if err != nil || len(peers) != 1 {
		t.Fatalf("paired devices = %v (%v), want one", peers, err)
	}
	g.pairedMu.Lock()
	g.paired = map[netip.AddrPort]trust.Peer{}
	g.pairedMu.Unlock()
	if address {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		gone := netip.MustParseAddrPort(ln.Addr().String())
		_ = ln.Close()
		g.rememberAddress(gone, peers[0].Key)
	}
	g.refreshDevices()
	items := pairedItems(g)
	if len(items) != 1 {
		t.Fatalf("paired section = %+v, want one device", items)
	}
	return items[0].row
}

// tapButton taps the button with the given text.
func tapButton(t *testing.T, obj fyne.CanvasObject, text string) {
	t.Helper()
	switch o := obj.(type) {
	case *widget.Button:
		if o.Text == text {
			test.Tap(o)
			return
		}
	case *container.Scroll:
		tapButton(t, o.Content, text)
	case *fyne.Container:
		for _, child := range o.Objects {
			if contains(buttonLabels(child), text) {
				tapButton(t, child, text)
				return
			}
		}
	}
}

// Regression: a paired device that is offline could not be unpaired from the
// home list. Opening it dialled the device first, the dial failed, and the
// device screen with the Unpair action never appeared. The iPhone and iPad app
// opens the device screen from the paired list, so this must too.
func TestOfflinePairedDeviceOpensItsScreenWithUnpair(t *testing.T) {
	g := newTestGUI(t)
	row := offlinePairedDevice(t, g, true)
	if row.Presence == discovery.PresenceOnline {
		t.Fatalf("the device must be offline for this test")
	}

	g.openDevice(row)
	waitForButton(t, g, g.tr.T("gui.trusted.remove"))

	// The screen shows who the device is, from the trust store.
	texts := labelTexts(g.win.Content())
	if !contains(texts, "other") {
		t.Fatalf("device screen does not show the paired name; texts = %v", texts)
	}
	for _, s := range texts {
		if s == "—" || s == g.tr.T("gui.device.paired_at", "—") {
			t.Fatalf("device screen shows a placeholder instead of the details: %q", s)
		}
	}

	// Tapping Unpair asks for confirmation.
	tapButton(t, g.win.Content(), g.tr.T("gui.trusted.remove"))
	if g.win.Canvas().Overlays().Top() == nil {
		t.Fatalf("Unpair on the device screen did not ask for confirmation")
	}
}

// A paired device with no known address is offline too, and can still be
// unpaired from its screen.
func TestPairedDeviceWithoutAddressOpensItsScreenWithUnpair(t *testing.T) {
	g := newTestGUI(t)
	row := offlinePairedDevice(t, g, false)
	if row.Address != "" {
		t.Fatalf("address = %q, want none", row.Address)
	}

	g.openDevice(row)
	waitForButton(t, g, g.tr.T("gui.trusted.remove"))

	tapButton(t, g.win.Content(), g.tr.T("gui.trusted.remove"))
	if g.win.Canvas().Overlays().Top() == nil {
		t.Fatalf("Unpair on the device screen did not ask for confirmation")
	}
}

// Regression: the same device paired twice, under two names, was listed twice
// in the paired section.
func TestHomeListsOneLinePerDeviceAddress(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	_ = pairedDevice(t, g)
	peers, err := g.store.List()
	if err != nil || len(peers) != 2 {
		t.Fatalf("paired devices = %v (%v), want two", peers, err)
	}
	if err := g.store.Rename(peers[0].Fingerprint, "Antigo"); err != nil {
		t.Fatal(err)
	}
	for _, p := range peers {
		g.rememberAddress(ep, p.Key)
	}
	g.refreshDevices()

	if n := len(pairedItems(g)); n != 1 {
		t.Fatalf("paired lines = %d (%+v), want 1", n, pairedItems(g))
	}
}
