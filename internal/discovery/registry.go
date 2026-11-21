package discovery

import (
	"fmt"
	"net/netip"
	"sort"
	"sync"
	"time"

	"tomorrow-drop/internal/apperr"
)

// Announcement is one accepted answer from a device.
type Announcement struct {
	Name string
	Addr netip.AddrPort
	Seen time.Time
}

// Device is an entry of the ephemeral list of discovered devices.
type Device struct {
	Name     string
	Addr     netip.AddrPort
	LastSeen time.Time
	// Trusted is set after an authenticated connection proves the device is trusted.
	Trusted bool
	// Pinned entries never expire while the application runs (see Pin).
	Pinned bool
}

// Registry is an ephemeral list of discovered devices. Entries expire after
// ttl without an answer, and the list holds at most capacity entries. Nothing
// is persisted.
type Registry struct {
	mu       sync.Mutex
	ttl      time.Duration
	capacity int
	devices  map[netip.AddrPort]*Device
}

// NewRegistry creates an empty registry.
func NewRegistry(ttl time.Duration, capacity int) *Registry {
	return &Registry{ttl: ttl, capacity: capacity, devices: map[netip.AddrPort]*Device{}}
}

// Observe adds or refreshes the device that sent the announcement.
func (r *Registry) Observe(a Announcement, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.devices[a.Addr]; ok {
		d.Name = a.Name
		d.LastSeen = a.Seen
		return nil
	}
	if len(r.devices) >= r.capacity {
		r.pruneLocked(now)
	}
	if len(r.devices) >= r.capacity {
		return fmt.Errorf("discovered device list is full: %w", apperr.ErrLimitExceeded)
	}
	r.devices[a.Addr] = &Device{Name: a.Name, Addr: a.Addr, LastSeen: a.Seen}
	return nil
}

// Pin adds the device at addr to the list, or keeps the entry it already has,
// and keeps it there without answers until Unpin. It is used for a device the
// person connected to by address: such a device may be on a network whose
// announcements never reach this one. Nothing is persisted, so a pinned entry
// lasts until the application closes.
func (r *Registry) Pin(addr netip.AddrPort, name string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.devices[addr]; ok {
		d.Pinned = true
		if name != "" {
			d.Name = name
		}
		return nil
	}
	if len(r.devices) >= r.capacity {
		r.pruneLocked(now)
	}
	if len(r.devices) >= r.capacity {
		return fmt.Errorf("discovered device list is full: %w", apperr.ErrLimitExceeded)
	}
	r.devices[addr] = &Device{Name: name, Addr: addr, LastSeen: now, Pinned: true}
	return nil
}

// Unpin gives the entry at addr back the normal expiry rule, if it is listed.
func (r *Registry) Unpin(addr netip.AddrPort) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.devices[addr]; ok {
		d.Pinned = false
	}
}

// Prune removes expired entries and returns how many were removed.
func (r *Registry) Prune(now time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pruneLocked(now)
}

// List returns the live devices sorted by name, then address.
func (r *Registry) List(now time.Time) []Device {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Device
	for _, d := range r.devices {
		if !r.expired(d, now) {
			out = append(out, *d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Addr.String() < out[j].Addr.String()
	})
	return out
}

// MarkTrusted sets the trusted flag of the device at addr, if it is listed.
func (r *Registry) MarkTrusted(addr netip.AddrPort, trusted bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.devices[addr]; ok {
		d.Trusted = trusted
	}
}

func (r *Registry) expired(d *Device, now time.Time) bool {
	if d.Pinned {
		return false
	}
	return now.Sub(d.LastSeen) > r.ttl
}

func (r *Registry) pruneLocked(now time.Time) int {
	removed := 0
	for addr, d := range r.devices {
		if r.expired(d, now) {
			delete(r.devices, addr)
			removed++
		}
	}
	return removed
}
