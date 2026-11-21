package main

import (
	"net/netip"

	"tomorrow-drop/internal/protocol"
)

// presenceTTL is how long a reachability check counts as current: two discovery
// rounds and the probe timeout. One missed round then shows as unknown, not as
// a stale "online".
const presenceTTL = 2*protocol.DiscoveryInterval + protocol.PresenceTimeout

// startPresenceCheck checks the paired devices in the background, unless a check
// is already running. It reports whether it started one. It never blocks, so
// it is safe to call from the discovery timer.
func (g *gui) startPresenceCheck() bool {
	if !g.presenceBusy.CompareAndSwap(false, true) {
		return false
	}
	g.background(func() {
		defer g.presenceBusy.Store(false)
		g.checkPresence()
	})
	return true
}

// checkPresence probes the last known address of every paired device, then
// redraws the home list.
func (g *gui) checkPresence() {
	addrs := g.pairedEndpoints()
	if len(addrs) == 0 {
		return
	}
	g.presence.Check(g.ctx, addrs, protocol.PresenceTimeout)
	g.refreshDevices()
}

// pairedEndpoints returns the last known address of each paired device. A
// device whose address is not known is not checked.
func (g *gui) pairedEndpoints() []netip.AddrPort {
	peers, err := g.store.List()
	if err != nil {
		return nil
	}
	addrs := g.pairedAddresses()
	var out []netip.AddrPort
	for _, p := range peers {
		text, ok := addrs[p.Fingerprint.Hex()]
		if !ok {
			continue
		}
		if ep, err := manualEndpoint(text); err == nil {
			out = append(out, ep)
		}
	}
	return out
}
