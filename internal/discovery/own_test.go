package discovery

import (
	"net/netip"
	"testing"
	"time"
)

// A device answers its own DISCOVER through the loopback of its interface, so
// the announcement comes from one of its own addresses on its own TCP port.
func TestIsOwnAnnouncementMatchesLocalAddressAndPort(t *testing.T) {
	local := []netip.Addr{netip.MustParseAddr("192.168.1.10")}
	own := Announcement{Name: "ipad", Addr: netip.MustParseAddrPort("192.168.1.10:47521"), Seen: time.Now()}
	if !IsOwnAnnouncement(own, local, 47521) {
		t.Fatalf("announcement from this device's own address and port was not recognized")
	}
}

func TestIsOwnAnnouncementKeepsOtherDevices(t *testing.T) {
	local := []netip.Addr{netip.MustParseAddr("192.168.1.10")}
	cases := map[string]netip.AddrPort{
		"other address, same port": netip.MustParseAddrPort("192.168.1.11:47521"),
		"own address, other port":  netip.MustParseAddrPort("192.168.1.10:47600"),
	}
	for name, addr := range cases {
		a := Announcement{Name: "phone", Addr: addr, Seen: time.Now()}
		if IsOwnAnnouncement(a, local, 47521) {
			t.Errorf("%s: another device was taken for this one", name)
		}
	}
}

// Interface addresses may come back as IPv4-mapped IPv6; they must still match.
func TestIsOwnAnnouncementAcceptsMappedAddresses(t *testing.T) {
	local := []netip.Addr{netip.MustParseAddr("::ffff:192.168.1.10")}
	a := Announcement{Name: "ipad", Addr: netip.MustParseAddrPort("192.168.1.10:47521"), Seen: time.Now()}
	if !IsOwnAnnouncement(a, local, 47521) {
		t.Fatalf("mapped local address did not match the announcement")
	}
}
