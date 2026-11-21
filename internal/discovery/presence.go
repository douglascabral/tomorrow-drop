package discovery

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"time"
)

// PresenceState tells whether a paired device answered a recent check.
type PresenceState int

const (
	// PresenceUnknown means no result is current: never checked, or the last
	// result is older than the time-to-live.
	PresenceUnknown PresenceState = iota
	// PresenceOnline means something accepted a TCP connection at the address.
	PresenceOnline
	// PresenceOffline means the last check did not connect.
	PresenceOffline
)

// presenceConcurrency bounds how many devices are probed at the same time.
const presenceConcurrency = 8

type presenceEntry struct {
	online    bool
	checkedAt time.Time
}

// Presence keeps, in memory, the last reachability check of each paired device,
// by the address it answered at. Nothing is persisted.
//
// A check is a TCP connect to the address with no TLS and no data: it says that
// something accepts connections there, not that it is the same device. Nothing
// is authorized by it.
type Presence struct {
	ttl time.Duration

	mu      sync.Mutex
	entries map[netip.AddrPort]presenceEntry
}

// NewPresence creates an empty presence table. A result older than ttl is unknown.
func NewPresence(ttl time.Duration) *Presence {
	return &Presence{ttl: ttl, entries: map[netip.AddrPort]presenceEntry{}}
}

// Record stores the result of a check of addr made at now.
func (p *Presence) Record(addr netip.AddrPort, online bool, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries[addr] = presenceEntry{online: online, checkedAt: now}
}

// State returns the current presence of addr at now.
func (p *Presence) State(addr netip.AddrPort, now time.Time) PresenceState {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.entries[addr]
	if !ok || now.Sub(e.checkedAt) > p.ttl {
		return PresenceUnknown
	}
	if e.online {
		return PresenceOnline
	}
	return PresenceOffline
}

// Forget drops the result for addr, for example when the device is unpaired.
func (p *Presence) Forget(addr netip.AddrPort) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.entries, addr)
}

// Check probes every address in parallel, at most presenceConcurrency at a
// time, and records each result. A cancelled ctx stops the check and records
// nothing, so an interrupted round never reports devices as offline.
func (p *Presence) Check(ctx context.Context, addrs []netip.AddrPort, timeout time.Duration) {
	type result struct {
		addr   netip.AddrPort
		online bool
	}
	results := make(chan result, len(addrs))
	sem := make(chan struct{}, presenceConcurrency)
	var wg sync.WaitGroup
	for _, addr := range addrs {
		wg.Add(1)
		go func(addr netip.AddrPort) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			results <- result{addr: addr, online: Probe(ctx, addr, timeout)}
		}(addr)
	}
	wg.Wait()
	close(results)
	if ctx.Err() != nil {
		return
	}
	now := time.Now()
	for r := range results {
		p.Record(r.addr, r.online, now)
	}
}

// Probe reports whether something accepts a TCP connection at addr within timeout.
// The connection is closed at once; nothing is sent.
func Probe(ctx context.Context, addr netip.AddrPort, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
