package main

import (
	"context"
	"net"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

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
	t.Cleanup(cancel)

	other, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	otherStore := trust.Open(filepath.Join(t.TempDir(), "trust.json"))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go answerLikeAPhone(ctx, ln, other, otherStore)
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
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
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
