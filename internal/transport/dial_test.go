package transport

import (
	"net"
	"net/netip"
	"testing"

	"tomorrow-drop/internal/protocol"
)

func TestDialManualEndpointUsesSameTLSAuthentication(t *testing.T) {
	ctx := testContext(t)
	server := newID(t)
	client := newID(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	serverDone := make(chan *Session, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverDone <- nil
			return
		}
		s, _ := Server(ctx, conn, server, newTrustStore(t))
		serverDone <- s
	}()

	endpoint := netip.MustParseAddrPort(ln.Addr().String())
	cs, err := Dial(ctx, endpoint, client, newTrustStore(t))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	ss := <-serverDone
	if ss == nil {
		t.Fatalf("server side failed")
	}
	if !cs.PeerKey().Equal(server.PublicKey()) {
		t.Fatalf("manual connection did not authenticate the server key")
	}
	if cs.State() != protocol.StateAwaitingVerification {
		t.Fatalf("state = %s, want awaiting_verification", cs.State())
	}
}

func TestDialFailsWhenNothingListens(t *testing.T) {
	ctx := testContext(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := netip.MustParseAddrPort(ln.Addr().String())
	_ = ln.Close()

	if _, err := Dial(ctx, endpoint, newID(t), newTrustStore(t)); err == nil {
		t.Fatalf("Dial to a closed port succeeded")
	}
}
