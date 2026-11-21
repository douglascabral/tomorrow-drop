package discovery

import "net/netip"

// IsOwnAnnouncement reports whether an announcement was sent by this device.
// A device hears its own DISCOVER on its interface, so its answer comes from
// one of its own addresses and announces the TCP port it listens on. Another
// device on the same address and port cannot exist, so the match is exact.
func IsOwnAnnouncement(a Announcement, local []netip.Addr, tcpPort int) bool {
	if int(a.Addr.Port()) != tcpPort {
		return false
	}
	from := a.Addr.Addr().Unmap()
	for _, addr := range local {
		if addr.Unmap() == from {
			return true
		}
	}
	return false
}
