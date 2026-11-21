package trust

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
)

// Role identifies each side's role in the TLS connection.
type Role byte

const (
	// Initiator is the side that opened the connection (TLS client).
	Initiator Role = 1
	// Responder is the side that accepted the connection (TLS server).
	Responder Role = 2
)

const (
	transcriptLabel = "tomorrow-drop pairing v1"
	bindingSize     = 32
)

func (r Role) opposite() Role {
	if r == Initiator {
		return Responder
	}
	return Initiator
}

// Pairing drives the mutual confirmation of a pairing (PRD §9).
//
// Each side produces an HMAC-SHA256 confirmation over the session's TLS
// exporter (a binding unique to each connection), its role, both public keys
// and the protocol version. The confirmation is therefore valid only for this
// session and this pair of keys. A side reveals its confirmation only after
// ConfirmLocal, which must be an explicit user action.
type Pairing struct {
	role      Role
	local     ed25519.PublicKey
	remote    ed25519.PublicKey
	binding   []byte
	version   protocol.Version
	confirmed bool
	verified  bool
}

// NewPairing creates the pairing state for a session.
func NewPairing(role Role, local, remote ed25519.PublicKey, binding []byte, version protocol.Version) (*Pairing, error) {
	if role != Initiator && role != Responder {
		return nil, fmt.Errorf("invalid pairing role: %w", apperr.ErrMalformed)
	}
	if len(local) != ed25519.PublicKeySize || len(remote) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key with invalid size: %w", apperr.ErrMalformed)
	}
	if len(binding) != bindingSize {
		return nil, fmt.Errorf("session binding of %d bytes: %w", len(binding), apperr.ErrMalformed)
	}
	if bytes.Equal(local, remote) {
		return nil, fmt.Errorf("pairing with own identity: %w", apperr.ErrMalformed)
	}
	return &Pairing{
		role:    role,
		local:   local,
		remote:  remote,
		binding: append([]byte(nil), binding...),
		version: version,
	}, nil
}

// ConfirmLocal records the user's explicit confirmation that the
// fingerprints were compared.
func (p *Pairing) ConfirmLocal() {
	p.confirmed = true
}

// LocalTag returns this side's confirmation. It exists only after ConfirmLocal.
func (p *Pairing) LocalTag() ([]byte, error) {
	if !p.confirmed {
		return nil, fmt.Errorf("missing local confirmation: %w", apperr.ErrUnauthorized)
	}
	return p.tag(p.role, p.local, p.remote), nil
}

// VerifyRemote validates the confirmation received from the peer in this session.
func (p *Pairing) VerifyRemote(tag []byte) error {
	expected := p.tag(p.role.opposite(), p.remote, p.local)
	if !hmac.Equal(tag, expected) {
		return fmt.Errorf("invalid peer confirmation: %w", apperr.ErrUnauthorized)
	}
	p.verified = true
	return nil
}

// Complete reports whether mutual confirmation was proven.
func (p *Pairing) Complete() bool {
	return p.confirmed && p.verified
}

func (p *Pairing) tag(role Role, confirmer, other ed25519.PublicKey) []byte {
	mac := hmac.New(sha256.New, p.binding)
	mac.Write([]byte(transcriptLabel))
	mac.Write([]byte{byte(role)})
	var version [2]byte
	binary.BigEndian.PutUint16(version[:], uint16(p.version))
	mac.Write(version[:])
	mac.Write(confirmer)
	mac.Write(other)
	return mac.Sum(nil)
}

// CommitPairing stores the peer's key as trusted. It refuses any pairing
// without proven mutual confirmation.
func CommitPairing(s *Store, p *Pairing, name string) error {
	if !p.Complete() {
		return fmt.Errorf("mutual confirmation incomplete: %w", apperr.ErrUnauthorized)
	}
	return s.add(p.remote, name)
}
