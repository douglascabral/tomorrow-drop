package ui

import (
	"net/netip"
	"testing"
)

// The home screen shows this device's address so two devices can be told
// apart on sight. Only addresses another device can reach are shown.
func TestLocalAddressTextShowsReachableIPv4Only(t *testing.T) {
	addrs := []netip.Addr{
		netip.MustParseAddr("127.0.0.1"),
		netip.MustParseAddr("169.254.10.3"),
		netip.MustParseAddr("fe80::1"),
		netip.MustParseAddr("192.168.1.10"),
		netip.MustParseAddr("::ffff:10.0.0.7"),
	}
	if got, want := LocalAddressText(addrs), "10.0.0.7, 192.168.1.10"; got != want {
		t.Fatalf("LocalAddressText = %q, want %q", got, want)
	}
}

func TestLocalAddressTextIsEmptyWithoutNetwork(t *testing.T) {
	if got := LocalAddressText([]netip.Addr{netip.MustParseAddr("127.0.0.1")}); got != "" {
		t.Fatalf("LocalAddressText = %q, want empty", got)
	}
}
