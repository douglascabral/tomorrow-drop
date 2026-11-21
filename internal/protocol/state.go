package protocol

// PeerState é o estado de segurança de um par, conforme PRD v1.9, seção 10.
// Uma conexão TLS estabelecida não implica StateTrusted.
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
