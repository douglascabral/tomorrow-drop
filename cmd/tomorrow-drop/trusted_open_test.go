package main

import (
	"context"
	"net"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/pairing"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

// Regression: opening a device that is already trusted must show its actions
// (send, receive), not the "already trusted" alert. The discovered row does not
// know about trust, so the device has to be contacted first.
func TestOpeningATrustedDeviceShowsItsActions(t *testing.T) {
	g := newTestGUI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	other, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	otherStore := trust.Open(filepath.Join(t.TempDir(), "trust.json"))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go answerLikeAPhone(ctx, ln, other, otherStore)
	ep := netip.MustParseAddrPort(ln.Addr().String())

	// Pair once, as a person does on first contact.
	s, err := transport.Dial(ctx, ep, g.id, g.store)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := pairing.Run(ctx, s, g.id, g.store, yesPrompts("other")); err != nil {
		t.Fatalf("pairing: %v", err)
	}
	_ = s.Close()

	// The list row does not know the device is trusted (only an authenticated
	// connection proves it, docs/DISCOVERY.md), so Trusted is false here.
	g.openDevice(ui.DeviceRow{Name: "other", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))
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
