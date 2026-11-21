package discovery

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
)

func TestBroadcastTargetsCoverEachIPv4Subnet(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.IPv4(192, 168, 1, 10), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.IPv4(10, 0, 0, 7), Mask: net.CIDRMask(16, 32)},
		&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.CIDRMask(8, 32)},
		&net.IPNet{IP: net.ParseIP("fd00::1"), Mask: net.CIDRMask(64, 128)},
	}
	targets := BroadcastTargets(7000, addrs)
	want := map[string]bool{
		"192.168.1.255:7000":   true,
		"10.0.255.255:7000":    true,
		"255.255.255.255:7000": true,
	}
	got := map[string]bool{}
	for _, tgt := range targets {
		got[tgt.String()] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("missing broadcast target %s (got %v)", w, targets)
		}
	}
	if got["127.255.255.255:7000"] {
		t.Errorf("loopback subnet should not be a discovery target")
	}
}

func TestScannerWithoutTargetsReportsNoRoute(t *testing.T) {
	sc, err := NewScanner(nil)
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}
	defer func() { _ = sc.Close() }()
	if err := sc.Query(time.Now()); !errors.Is(err, apperr.ErrNoRoute) {
		t.Fatalf("Query without targets = %v, want ErrNoRoute", err)
	}
}

func TestScannerFindsListenerOverLoopback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln, err := Listen(ctx, ListenerOptions{Addr: "127.0.0.1:0", Name: "desk", TCPPort: 9001})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	sc, err := NewScanner([]netip.AddrPort{ln.LocalAddr()})
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}
	defer func() { _ = sc.Close() }()

	found := make(chan Announcement, 4)
	go func() { _ = sc.Run(ctx, found) }()

	if err := sc.Query(time.Now()); err != nil {
		t.Fatalf("Query: %v", err)
	}
	select {
	case a := <-found:
		if a.Name != "desk" || a.Addr.Port() != 9001 {
			t.Fatalf("announcement = %+v, want desk on port 9001", a)
		}
	case <-ctx.Done():
		t.Fatalf("no announcement received")
	}
}

func TestScannerIgnoresAnnouncementWithUnknownNonce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sc, err := NewScanner(nil)
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}
	defer func() { _ = sc.Close() }()
	found := make(chan Announcement, 4)
	go func() { _ = sc.Run(ctx, found) }()

	// Unsolicited announcement: the nonce was never sent by this scanner.
	forged, _ := NewDiscover()
	answer, _ := NewAnnounce(forged, "intruder", 7000)
	data, _ := Encode(answer)
	sendRaw(t, sc.LocalAddr(), data)

	select {
	case a := <-found:
		t.Fatalf("unsolicited announcement accepted: %+v", a)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestScannerExpiresPendingNonces(t *testing.T) {
	sc, err := NewScanner([]netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:9")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sc.Close() }()
	now := time.Unix(1000, 0)
	if err := sc.Query(now); err != nil {
		t.Fatalf("Query: %v", err)
	}
	if n := sc.pendingCount(now.Add(pendingTTL + time.Second)); n != 0 {
		t.Fatalf("expired nonces still pending: %d", n)
	}
}

// sendRaw sends data from a fresh UDP socket to dst, bypassing the scanner's own logic.
func sendRaw(t *testing.T, dst netip.AddrPort, data []byte) {
	t.Helper()
	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(dst))
	if err != nil {
		t.Fatalf("dial udp: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
}
