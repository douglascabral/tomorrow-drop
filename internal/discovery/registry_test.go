package discovery

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
)

func endpoint(s string) netip.AddrPort { return netip.MustParseAddrPort(s) }

func TestRegistryObserveAddsAndListsDevices(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	if err := r.Observe(Announcement{Name: "laptop", Addr: endpoint("10.0.0.2:7000"), Seen: now}, now); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	devices := r.List(now)
	if len(devices) != 1 || devices[0].Name != "laptop" || devices[0].Trusted {
		t.Fatalf("devices = %+v, want one untrusted laptop", devices)
	}
}

func TestRegistryUpdatesExistingEntry(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	addr := endpoint("10.0.0.2:7000")
	_ = r.Observe(Announcement{Name: "old", Addr: addr, Seen: now}, now)
	later := now.Add(30 * time.Second)
	_ = r.Observe(Announcement{Name: "new", Addr: addr, Seen: later}, later)

	devices := r.List(later)
	if len(devices) != 1 || devices[0].Name != "new" || !devices[0].LastSeen.Equal(later) {
		t.Fatalf("entry not updated: %+v", devices)
	}
}

func TestRegistryExpiresStaleEntries(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	_ = r.Observe(Announcement{Name: "gone", Addr: endpoint("10.0.0.3:7000"), Seen: now}, now)

	if got := len(r.List(now.Add(89 * time.Second))); got != 1 {
		t.Fatalf("entry expired too early: %d devices", got)
	}
	if got := len(r.List(now.Add(91 * time.Second))); got != 0 {
		t.Fatalf("stale entry still listed: %d devices", got)
	}
	if removed := r.Prune(now.Add(91 * time.Second)); removed != 1 {
		t.Fatalf("Prune removed %d entries, want 1", removed)
	}
}

func TestRegistryRespectsCapacity(t *testing.T) {
	r := NewRegistry(90*time.Second, 2)
	now := time.Unix(1000, 0)
	_ = r.Observe(Announcement{Name: "a", Addr: endpoint("10.0.0.1:7000"), Seen: now}, now)
	_ = r.Observe(Announcement{Name: "b", Addr: endpoint("10.0.0.2:7000"), Seen: now}, now)
	err := r.Observe(Announcement{Name: "c", Addr: endpoint("10.0.0.3:7000"), Seen: now}, now)
	if !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Fatalf("Observe above capacity = %v, want ErrLimitExceeded", err)
	}
}

func TestRegistryMarksTrustedDevices(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	addr := endpoint("10.0.0.2:7000")
	_ = r.Observe(Announcement{Name: "phone", Addr: addr, Seen: now}, now)
	r.MarkTrusted(addr, true)
	if devices := r.List(now); !devices[0].Trusted {
		t.Fatalf("device not marked as trusted")
	}
}

func TestRegistryListIsSortedByName(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	_ = r.Observe(Announcement{Name: "zeta", Addr: endpoint("10.0.0.1:7000"), Seen: now}, now)
	_ = r.Observe(Announcement{Name: "alpha", Addr: endpoint("10.0.0.2:7000"), Seen: now}, now)
	devices := r.List(now)
	if devices[0].Name != "alpha" || devices[1].Name != "zeta" {
		t.Fatalf("order = %s, %s; want alpha, zeta", devices[0].Name, devices[1].Name)
	}
}

// A device paired by address may be on a network the announcements do not
// reach (another VLAN). Once it is pinned it stays listed while the application
// runs, even when nothing answers from it.
func TestRegistryPinnedDeviceDoesNotExpire(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	addr := endpoint("10.9.0.5:47521")
	if err := r.Pin(addr, "office", now); err != nil {
		t.Fatalf("Pin: %v", err)
	}
	later := now.Add(time.Hour)
	devices := r.List(later)
	if len(devices) != 1 || devices[0].Name != "office" || devices[0].Addr != addr {
		t.Fatalf("pinned device not listed after expiry time: %+v", devices)
	}
	if removed := r.Prune(later); removed != 0 {
		t.Fatalf("Prune removed %d pinned entries, want 0", removed)
	}
}

// A pinned device that announces again keeps its pin, so a later Observe does
// not put it back under the expiry rule.
func TestRegistryObserveKeepsThePin(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	addr := endpoint("10.9.0.5:47521")
	_ = r.Pin(addr, "office", now)
	later := now.Add(10 * time.Minute)
	_ = r.Observe(Announcement{Name: "office-pc", Addr: addr, Seen: later}, later)
	if got := len(r.List(later.Add(time.Hour))); got != 1 {
		t.Fatalf("pinned device expired after an announcement: %d devices", got)
	}
}

// Unpinning gives the entry back the normal expiry rule.
func TestRegistryUnpinnedDeviceExpiresAgain(t *testing.T) {
	r := NewRegistry(90*time.Second, 8)
	now := time.Unix(1000, 0)
	addr := endpoint("10.9.0.5:47521")
	_ = r.Pin(addr, "office", now)
	r.Unpin(addr)
	if got := len(r.List(now.Add(91 * time.Second))); got != 0 {
		t.Fatalf("unpinned device still listed after its expiry time: %d devices", got)
	}
}
