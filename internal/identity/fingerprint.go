package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"tomorrow-drop/internal/apperr"
)

// Fingerprint is the SHA-256 of the 32 canonical bytes of an Ed25519 public key (PRD §7).
type Fingerprint [sha256.Size]byte

// FingerprintOf computes the fingerprint of an Ed25519 public key.
func FingerprintOf(pub ed25519.PublicKey) (Fingerprint, error) {
	if len(pub) != ed25519.PublicKeySize {
		return Fingerprint{}, fmt.Errorf("public key of %d bytes: %w", len(pub), apperr.ErrMalformed)
	}
	return Fingerprint(sha256.Sum256(pub)), nil
}

// Hex returns the 64 hexadecimal characters, without truncation.
func (f Fingerprint) Hex() string {
	return hex.EncodeToString(f[:])
}

// Grouped returns the fingerprint in groups of 4 characters separated by spaces.
func (f Fingerprint) Grouped() string {
	h := f.Hex()
	groups := make([]string, 0, len(h)/4)
	for i := 0; i < len(h); i += 4 {
		groups = append(groups, h[i:i+4])
	}
	return strings.Join(groups, " ")
}

func (f Fingerprint) String() string {
	return f.Grouped()
}
