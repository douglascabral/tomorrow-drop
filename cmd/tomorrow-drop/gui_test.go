package main

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
)

func newTestGUI(t *testing.T) *gui {
	t.Helper()
	tr, err := i18n.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	home := t.TempDir()
	g, err := newGUI(ctx, tr, home, filepath.Join(home, "settings.json"))
	if err != nil {
		t.Fatalf("newGUI: %v", err)
	}
	g.app = test.NewApp()
	g.win = g.app.NewWindow("test")
	return g
}

func TestHomeShowsDiscoveredDevicesFromRegistry(t *testing.T) {
	g := newTestGUI(t)
	addr := netip.MustParseAddrPort("10.0.0.7:47521")
	if err := g.reg.Observe(discovery.Announcement{Name: "living-room", Addr: addr, Seen: time.Now()}, time.Now()); err != nil {
		t.Fatal(err)
	}
	g.showHome()
	if len(g.rows) != 1 || g.rows[0].Name != "living-room" || g.rows[0].Address != addr.String() {
		t.Fatalf("home rows = %+v, want the discovered device", g.rows)
	}
	if g.rows[0].Trusted {
		t.Fatalf("a discovered device must not be shown as trusted")
	}
}

// Sending to a device that is not trusted must never reach the transfer code.
func TestSendRefusesDeviceThatIsNotTrusted(t *testing.T) {
	g := newTestGUI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	other, _ := identity.Generate()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		s, err := transport.Server(ctx, conn, other, trust.Open(filepath.Join(t.TempDir(), "t.json")))
		if err == nil {
			defer func() { _ = s.Close() }()
			<-ctx.Done()
		}
	}()

	spec := transfer.SendSpec{Kind: transfer.KindText, Text: "secret"}
	err = g.sendTo(ctx, ln.Addr().String(), spec, nil)
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("sendTo an untrusted device = %v, want ErrUnauthorized", err)
	}
}

func TestEndpointForRejectsUnknownName(t *testing.T) {
	g := newTestGUI(t)
	if _, err := g.endpointFor("nobody-here"); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("endpointFor unknown name = %v, want ErrMalformed", err)
	}
}

// Regression: returning to the foreground must send a query at once (PRD
// §5.2), even when the window was left and reopened within the refresh
// cooldown. Using Refresh there suppressed that query.
func TestForegroundSendsQueryEvenWithinRefreshCooldown(t *testing.T) {
	g := newTestGUI(t)
	g.tlsPort = 0
	g.discPort = 0

	g.setAvailable(true)
	g.setAvailable(false)
	g.setAvailable(true)
	g.setAvailable(false)

	if n := g.queries.Load(); n != 2 {
		t.Fatalf("queries after two foreground events = %d, want 2", n)
	}
}
