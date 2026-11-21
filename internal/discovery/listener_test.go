package discovery

import (
	"context"
	"net"
	"testing"
	"time"
)

func startListener(t *testing.T, opts ListenerOptions) (*Listener, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ln, err := Listen(ctx, opts)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	return ln, cancel
}

// queryAndCollect sends n DISCOVER datagrams from one socket and returns how
// many ANNOUNCE replies came back within the wait time.
func queryAndCollect(t *testing.T, dst *net.UDPAddr, n int, wait time.Duration) int {
	t.Helper()
	conn, err := net.DialUDP("udp", nil, dst)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	for i := 0; i < n; i++ {
		q, _ := NewDiscover()
		data, _ := Encode(q)
		if _, err := conn.Write(data); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	replies := 0
	buf := make([]byte, 2048)
	deadline := time.Now().Add(wait)
	for {
		_ = conn.SetReadDeadline(deadline)
		if _, err := conn.Read(buf); err != nil {
			return replies
		}
		replies++
	}
}

func TestListenerRespondsToDiscover(t *testing.T) {
	ln, _ := startListener(t, ListenerOptions{Addr: "127.0.0.1:0", Name: "desk", TCPPort: 9001})
	if got := queryAndCollect(t, net.UDPAddrFromAddrPort(ln.LocalAddr()), 1, time.Second); got != 1 {
		t.Fatalf("replies = %d, want 1", got)
	}
}

func TestListenerEnforcesPerOriginRateLimit(t *testing.T) {
	ln, _ := startListener(t, ListenerOptions{
		Addr: "127.0.0.1:0", Name: "desk", TCPPort: 9001,
		RateLimit: 3, RateWindow: time.Minute,
	})
	if got := queryAndCollect(t, net.UDPAddrFromAddrPort(ln.LocalAddr()), 10, time.Second); got != 3 {
		t.Fatalf("replies = %d, want 3 (rate limit)", got)
	}
}

func TestListenerIgnoresMalformedDatagrams(t *testing.T) {
	ln, _ := startListener(t, ListenerOptions{Addr: "127.0.0.1:0", Name: "desk", TCPPort: 9001})
	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(ln.LocalAddr()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	for _, payload := range []string{"garbage", `{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"x","port":1}`} {
		if _, err := conn.Write([]byte(payload)); err != nil {
			t.Fatal(err)
		}
	}
	buf := make([]byte, 2048)
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if n, err := conn.Read(buf); err == nil {
		t.Fatalf("listener replied to a malformed datagram: %q", buf[:n])
	}
}

func TestListenerStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ln, err := Listen(ctx, ListenerOptions{Addr: "127.0.0.1:0", Name: "desk", TCPPort: 9001})
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.LocalAddr()

	cancel()
	time.Sleep(100 * time.Millisecond)
	if got := queryAndCollect(t, net.UDPAddrFromAddrPort(addr), 1, 300*time.Millisecond); got != 0 {
		t.Fatalf("listener still replied after its context ended: %d", got)
	}
}
