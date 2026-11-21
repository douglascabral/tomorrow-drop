package main

import (
	"net"
	"net/netip"

	"tomorrow-drop/internal/transport"
)

// netAddr is the listener address shown to people and passed to Ready.
type netAddr = netip.AddrPort

func toNetAddr(a *net.TCPAddr) netAddr {
	return a.AddrPort()
}

// transportTrust is the trust lookup used by the server side.
type transportTrust = transport.TrustLookup
