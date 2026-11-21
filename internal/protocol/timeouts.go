package protocol

import "time"

const (
	// HandshakeTimeout limits the TLS handshake and version negotiation.
	HandshakeTimeout = 10 * time.Second
	// DecisionTimeout limits the wait for the recipient's accept or decline (PRD §13).
	DecisionTimeout = 60 * time.Second
	// IdleTimeout closes connections without traffic.
	IdleTimeout = 30 * time.Second
	// DiscoveryInterval is the DISCOVER interval while the app is visible (PRD §5.2).
	DiscoveryInterval = 30 * time.Second
	// PresenceTimeout limits the TCP connect that checks whether a paired device
	// is reachable (internal/discovery/presence.go). No TLS and no data are sent.
	PresenceTimeout = 2 * time.Second
)
