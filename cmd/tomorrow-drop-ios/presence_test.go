package main

import (
	"crypto/ed25519"
	"encoding/json"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/identity"
)

const (
	discoveryOnline  = discovery.PresenceOnline
	discoveryUnknown = discovery.PresenceUnknown
)

// trustRecord and trustFile mirror the trust list file (internal/trust), so a
// test can put a paired device in the list without running a pairing.
type trustRecord struct {
	Name        string    `json:"name"`
	PublicKey   []byte    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	AddedAt     time.Time `json:"added_at"`
}

type trustFile struct {
	Version int           `json:"version"`
	Peers   []trustRecord `json:"peers"`
}

// addPaired writes a paired device with a new key into the trust list of home.
func addPaired(t *testing.T, home, name string) identity.Fingerprint {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	fp := id.Fingerprint()
	path := filepath.Join(home, "trust.json")
	var f trustFile
	if data, err := os.ReadFile(path); err == nil { //nolint:gosec // G304: trust file inside the test's own temporary folder
		if err := json.Unmarshal(data, &f); err != nil {
			t.Fatal(err)
		}
	}
	f.Version = 1
	pub := ed25519.PublicKey(id.PublicKey())
	f.Peers = append(f.Peers, trustRecord{Name: name, PublicKey: pub, Fingerprint: fp.Hex(), AddedAt: time.Now()})
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return fp
}

// listenAt opens a TCP listener on loopback; it is closed at the end of the test.
func listenAt(t *testing.T) netip.AddrPort {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return netip.MustParseAddrPort(ln.Addr().String())
}

// closedAt returns an address where nothing listens.
func closedAt(t *testing.T) netip.AddrPort {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ep := netip.MustParseAddrPort(ln.Addr().String())
	_ = ln.Close()
	return ep
}

func newPresenceCore(t *testing.T) *core {
	t.Helper()
	home := t.TempDir()
	c, err := newCore(home, t.TempDir(), "iPad test", "en")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// A paired device is "unknown" until it has been checked.
func TestPairedDeviceIsUnknownUntilChecked(t *testing.T) {
	c := newPresenceCore(t)
	fp := addPaired(t, c.home, "Mac")
	c.mu.Lock()
	c.addrs[fp.Hex()] = listenAt(t).String()
	c.mu.Unlock()
	if !strings.Contains(c.toJSON(), `"presence":"unknown"`) {
		t.Fatalf("a device never checked is not unknown: %s", c.toJSON())
	}
}

// A paired device whose address accepts connections is online; once nothing
// answers there, it is offline.
func TestPairedDeviceReportsOnlineThenOffline(t *testing.T) {
	c := newPresenceCore(t)
	fp := addPaired(t, c.home, "Mac")
	up := listenAt(t)
	c.mu.Lock()
	c.addrs[fp.Hex()] = up.String()
	c.mu.Unlock()

	c.checkPresence()
	if !strings.Contains(c.toJSON(), `"presence":"online"`) {
		t.Fatalf("a device that accepts connections is not online: %s", c.toJSON())
	}

	c.mu.Lock()
	c.addrs[fp.Hex()] = closedAt(t).String()
	c.mu.Unlock()
	c.checkPresence()
	if !strings.Contains(c.toJSON(), `"presence":"offline"`) {
		t.Fatalf("a device at a closed address is not offline: %s", c.toJSON())
	}
}

// Online paired devices come first in the state the interface reads.
func TestOnlinePairedDevicesAreFirstInState(t *testing.T) {
	c := newPresenceCore(t)
	down := addPaired(t, c.home, "Aaa offline")
	up := addPaired(t, c.home, "Zzz online")
	c.mu.Lock()
	c.addrs[down.Hex()] = closedAt(t).String()
	c.addrs[up.Hex()] = listenAt(t).String()
	c.mu.Unlock()
	c.checkPresence()

	st := c.state()
	if len(st.Paired) != 2 || st.Paired[0].Name != "Zzz online" {
		t.Fatalf("paired order = %+v, want the online device first", st.Paired)
	}
}

// Unpairing forgets the reachability of the device, through the same bridge
// call the interface uses.
func TestUnpairForgetsPresence(t *testing.T) {
	c := newPresenceCore(t)
	fp := addPaired(t, c.home, "Mac")
	up := listenAt(t)
	c.mu.Lock()
	c.addrs[fp.Hex()] = up.String()
	c.mu.Unlock()
	c.checkPresence()
	if got := c.presence.State(up, time.Now()); got != discoveryOnline {
		t.Fatalf("before unpair the device is %v, want online", got)
	}

	peer, ok := c.pairedByHex(fp.Hex())
	if !ok {
		t.Fatalf("device not paired")
	}
	if err := c.unpair(peer); err != nil {
		t.Fatalf("unpair: %v", err)
	}
	if got := c.presence.State(up, time.Now()); got != discoveryUnknown {
		t.Fatalf("presence after unpair = %v, want unknown", got)
	}
	if n := len(c.state().Paired); n != 0 {
		t.Fatalf("paired devices after unpair = %d, want 0", n)
	}
}

// Regression: the same device paired twice, under two names, was listed twice.
func TestPairedListHasOneLinePerAddress(t *testing.T) {
	c := newPresenceCore(t)
	old := addPaired(t, c.home, "Mac antigo")
	time.Sleep(10 * time.Millisecond)
	cur := addPaired(t, c.home, "Mac")
	at := closedAt(t)
	c.mu.Lock()
	c.addrs[old.Hex()] = at.String()
	c.addrs[cur.Hex()] = at.String()
	c.mu.Unlock()

	st := c.state()
	if len(st.Paired) != 1 || st.Paired[0].Name != "Mac" {
		t.Fatalf("paired = %+v, want only the latest pairing, Mac", st.Paired)
	}
}
