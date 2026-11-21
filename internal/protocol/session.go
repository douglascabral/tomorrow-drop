package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// SessionID identifies a connection session. It is not an authentication
// credential: identity is proven by TLS and the Ed25519 key.
type SessionID [16]byte

// NewSessionID generates a random identifier with crypto/rand.
func NewSessionID() (SessionID, error) {
	var id SessionID
	if _, err := rand.Read(id[:]); err != nil {
		return SessionID{}, fmt.Errorf("generate session: %w", err)
	}
	return id, nil
}

// String returns the identifier in hexadecimal (32 characters).
func (id SessionID) String() string {
	return hex.EncodeToString(id[:])
}
