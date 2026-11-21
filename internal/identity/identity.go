// Package identity manages the local Ed25519 identity (PRD §6–7).
//
// The private key has no public accessor. Textual representations show only
// the fingerprint.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// Identity is the Ed25519 key pair of this device.
type Identity struct {
	priv ed25519.PrivateKey
}

// Generate creates a new identity with crypto/rand.
func Generate() (*Identity, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate Ed25519 key: %w", err)
	}
	return &Identity{priv: priv}, nil
}

// PublicKey returns a copy of the public key.
func (id *Identity) PublicKey() ed25519.PublicKey {
	// In ed25519.PrivateKey, the last 32 bytes are the public key.
	return append(ed25519.PublicKey(nil), id.priv[ed25519.PrivateKeySize-ed25519.PublicKeySize:]...)
}

// Fingerprint returns the SHA-256 of the public key.
func (id *Identity) Fingerprint() Fingerprint {
	return Fingerprint(sha256.Sum256(id.PublicKey()))
}

func (id *Identity) String() string {
	return "identity " + id.Fingerprint().Grouped()
}

func (id *Identity) GoString() string {
	return id.String()
}
