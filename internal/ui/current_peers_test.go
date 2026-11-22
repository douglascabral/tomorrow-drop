package ui

import (
	"testing"
	"time"

	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/trust"
)

func peerAt(t *testing.T, name string, added time.Time) trust.Peer {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return trust.Peer{Name: name, Key: id.PublicKey(), Fingerprint: id.Fingerprint(), AddedAt: added}
}

// Regression: the same device paired again (a new key) under another name was
// listed twice, because the earlier pairing stayed in the trust list with the
// same address. Only the latest pairing at an address is listed.
func TestCurrentPeersKeepsOnlyTheLatestPairingAtAnAddress(t *testing.T) {
	now := time.Now()
	old := peerAt(t, "Mac antigo", now.Add(-48*time.Hour))
	cur := peerAt(t, "Mac", now)
	other := peerAt(t, "Phone", now.Add(-time.Hour))
	addrs := map[string]string{
		old.Fingerprint.Hex():   "10.0.0.5:47521",
		cur.Fingerprint.Hex():   "10.0.0.5:47521",
		other.Fingerprint.Hex(): "10.0.0.6:47521",
	}
	got := CurrentPeers([]trust.Peer{old, cur, other}, addrs)
	if len(got) != 2 || got[0].Name != "Mac" || got[1].Name != "Phone" {
		t.Fatalf("CurrentPeers = %+v, want Mac and Phone", got)
	}
}

// Devices with no known address are different devices: nothing is merged.
func TestCurrentPeersKeepsDevicesWithoutAddress(t *testing.T) {
	now := time.Now()
	a := peerAt(t, "A", now)
	b := peerAt(t, "B", now.Add(-time.Hour))
	got := CurrentPeers([]trust.Peer{a, b}, map[string]string{})
	if len(got) != 2 {
		t.Fatalf("CurrentPeers = %+v, want both", got)
	}
}
