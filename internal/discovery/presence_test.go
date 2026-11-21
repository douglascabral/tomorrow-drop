package discovery

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"
)

// listenLocal opens a TCP listener on loopback and returns it with its address.
func listenLocal(t *testing.T) (net.Listener, netip.AddrPort) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln, netip.MustParseAddrPort(ln.Addr().String())
}

// closedAddr returns an address where nothing listens: a port opened and closed.
func closedAddr(t *testing.T) netip.AddrPort {
	t.Helper()
	ln, ep := listenLocal(t)
	_ = ln.Close()
	return ep
}

func TestProbeIsTrueWhenSomethingAcceptsTCP(t *testing.T) {
	_, ep := listenLocal(t)
	if !Probe(context.Background(), ep, 2*time.Second) {
		t.Fatalf("Probe to a listening port = false, want true")
	}
}

func TestProbeIsFalseWhenNothingListens(t *testing.T) {
	if Probe(context.Background(), closedAddr(t), 2*time.Second) {
		t.Fatalf("Probe to a closed port = true, want false")
	}
}

func TestPresenceIsUnknownUntilChecked(t *testing.T) {
	p := NewPresence(time.Minute)
	if got := p.State(endpoint("10.0.0.2:47521"), time.Unix(1000, 0)); got != PresenceUnknown {
		t.Fatalf("State of a device never checked = %v, want unknown", got)
	}
}

func TestPresenceRecordsOnlineAndOffline(t *testing.T) {
	p := NewPresence(time.Minute)
	addr := endpoint("10.0.0.2:47521")
	now := time.Unix(1000, 0)

	p.Record(addr, true, now)
	if got := p.State(addr, now.Add(10*time.Second)); got != PresenceOnline {
		t.Fatalf("State after a successful check = %v, want online", got)
	}
	p.Record(addr, false, now.Add(20*time.Second))
	if got := p.State(addr, now.Add(30*time.Second)); got != PresenceOffline {
		t.Fatalf("State after a failed check = %v, want offline", got)
	}
}

// A result older than the TTL is not shown as current: the device goes back to
// unknown, so one missed round never leaves a stale "online" on the screen.
func TestPresenceExpiresToUnknown(t *testing.T) {
	p := NewPresence(time.Minute)
	addr := endpoint("10.0.0.2:47521")
	now := time.Unix(1000, 0)
	p.Record(addr, true, now)

	if got := p.State(addr, now.Add(59*time.Second)); got != PresenceOnline {
		t.Fatalf("State inside the TTL = %v, want online", got)
	}
	if got := p.State(addr, now.Add(61*time.Second)); got != PresenceUnknown {
		t.Fatalf("State after the TTL = %v, want unknown", got)
	}
}

func TestPresenceForgetRemovesTheDevice(t *testing.T) {
	p := NewPresence(time.Minute)
	addr := endpoint("10.0.0.2:47521")
	now := time.Unix(1000, 0)
	p.Record(addr, true, now)
	p.Forget(addr)
	if got := p.State(addr, now); got != PresenceUnknown {
		t.Fatalf("State after Forget = %v, want unknown", got)
	}
}

// Check probes every address and records each result, online or offline.
func TestCheckRecordsEveryAddress(t *testing.T) {
	p := NewPresence(time.Minute)
	_, up := listenLocal(t)
	down := closedAddr(t)

	p.Check(context.Background(), []netip.AddrPort{up, down}, 2*time.Second)

	if got := p.State(up, time.Now()); got != PresenceOnline {
		t.Fatalf("listening device = %v, want online", got)
	}
	if got := p.State(down, time.Now()); got != PresenceOffline {
		t.Fatalf("closed device = %v, want offline", got)
	}
}

// A check that is cancelled says nothing about the devices: nothing is recorded.
func TestCheckWithCancelledContextRecordsNothing(t *testing.T) {
	p := NewPresence(time.Minute)
	_, up := listenLocal(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p.Check(ctx, []netip.AddrPort{up}, 2*time.Second)

	if got := p.State(up, time.Now()); got != PresenceUnknown {
		t.Fatalf("cancelled check recorded %v, want unknown", got)
	}
}
