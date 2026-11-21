package transport

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
)

// Dial opens a manual connection to a TCP endpoint. It uses the same TLS
// authentication as the automatic flow: no key is pinned, and the state is
// derived from the trust list after the handshake.
func Dial(ctx context.Context, endpoint netip.AddrPort, id *identity.Identity, trustList TrustLookup) (*Session, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", endpoint.String())
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", endpoint, contextOr(ctx, err))
	}
	s, err := Client(ctx, conn, id, nil, trustList)
	if err != nil {
		return nil, apperr.FromContext(err)
	}
	return s, nil
}
