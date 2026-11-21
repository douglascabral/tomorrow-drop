package discovery

import (
	"errors"
	"net"
	"net/netip"
	"testing"
)

// On Android, apps may not list network interfaces, so the address must come
// from the route when the listing fails (the phone showed no address before).
func TestLocalAddressesFallBackToRouteWhenListingFails(t *testing.T) {
	list := func() ([]net.Addr, error) { return nil, errors.New("netlinkrib: permission denied") }
	route := func() (netip.Addr, error) { return netip.MustParseAddr("10.27.27.111"), nil }
	got := localAddressesFrom(list, route)
	if len(got) != 1 || got[0].String() != "10.27.27.111" {
		t.Fatalf("addresses = %v, want the route address", got)
	}
}

func TestLocalAddressesUseTheListingWhenItWorks(t *testing.T) {
	list := func() ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{IP: net.ParseIP("192.168.1.9"), Mask: net.CIDRMask(24, 32)}}, nil
	}
	route := func() (netip.Addr, error) {
		t.Fatalf("route must not be asked when the listing works")
		return netip.Addr{}, nil
	}
	got := localAddressesFrom(list, route)
	if len(got) != 1 || got[0].String() != "192.168.1.9" {
		t.Fatalf("addresses = %v, want the listed address", got)
	}
}

func TestLocalAddressesEmptyWhenNothingWorks(t *testing.T) {
	list := func() ([]net.Addr, error) { return nil, errors.New("denied") }
	route := func() (netip.Addr, error) { return netip.Addr{}, errors.New("no route") }
	if got := localAddressesFrom(list, route); len(got) != 0 {
		t.Fatalf("addresses = %v, want none", got)
	}
}
