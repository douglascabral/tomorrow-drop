package discovery

import (
	"fmt"
	"net/netip"
	"strings"

	"tomorrow-drop/internal/apperr"
)

const maxEndpointLength = 64

// ParseEndpoint validates a manual IP:port entry. Host names are rejected, so
// no name resolution happens. Unspecified, multicast and broadcast addresses
// and port 0 are rejected.
func ParseEndpoint(input string) (netip.AddrPort, error) {
	s := strings.TrimSpace(input)
	if s == "" || len(s) > maxEndpointLength {
		return netip.AddrPort{}, fmt.Errorf("endpoint length: %w", apperr.ErrMalformed)
	}
	ep, err := netip.ParseAddrPort(s)
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("endpoint %q: %w: %w", s, apperr.ErrMalformed, err)
	}
	addr := ep.Addr()
	switch {
	case ep.Port() == 0:
		return netip.AddrPort{}, fmt.Errorf("port 0: %w", apperr.ErrMalformed)
	case addr.IsUnspecified(), addr.IsMulticast():
		return netip.AddrPort{}, fmt.Errorf("address %s is not a device address: %w", addr, apperr.ErrMalformed)
	case addr == netip.AddrFrom4([4]byte{255, 255, 255, 255}):
		return netip.AddrPort{}, fmt.Errorf("broadcast address: %w", apperr.ErrMalformed)
	}
	return ep, nil
}
