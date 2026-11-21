package protocol

// PeerState is the security state of a peer, per PRD v1.9, section 10.
// An established TLS connection does not imply StateTrusted.
type PeerState int

const (
	StateDiscovered PeerState = iota
	StateConnecting
	StateAwaitingVerification
	StateTrusted
	StateIdentityChanged
	StateDisconnected
)

var stateNames = [...]string{
	StateDiscovered:           "discovered",
	StateConnecting:           "connecting",
	StateAwaitingVerification: "awaiting_verification",
	StateTrusted:              "trusted",
	StateIdentityChanged:      "identity_changed",
	StateDisconnected:         "disconnected",
}

func (s PeerState) String() string {
	if s < 0 || int(s) >= len(stateNames) {
		return "unknown"
	}
	return stateNames[s]
}
