package ui

import (
	"net/netip"
	"sort"
	"strings"
)

// LocalAddressText lists the IPv4 addresses another device can reach this one
// at, sorted and separated by commas. Loopback and link-local addresses are
// left out; IPv6 is not shown.
func LocalAddressText(addrs []netip.Addr) string {
	var shown []string
	for _, a := range addrs {
		a = a.Unmap()
		if !a.Is4() || a.IsLoopback() || a.IsLinkLocalUnicast() || a.IsUnspecified() {
			continue
		}
		shown = append(shown, a.String())
	}
	sort.Strings(shown)
	return strings.Join(shown, ", ")
}
