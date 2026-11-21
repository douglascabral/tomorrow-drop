// Package pairing runs the mutual confirmation of a first contact (PRD §9).
// The command line and the graphical application both call Run, so the rules
// cannot diverge between interfaces.
package pairing

import (
	"context"
	"fmt"
	"strings"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
)

// ErrCancelled means the person did not confirm the comparison of fingerprints.
// It matches apperr.ErrUnauthorized, so interfaces report it as a refusal.
var ErrCancelled = fmt.Errorf("pairing not confirmed: %w", apperr.ErrUnauthorized)

// Prompts are the questions asked of the person. Confirm receives both
// complete fingerprints, so the person can compare them on both devices. Name
// is optional; without a name the first 8 hexadecimal characters are used.
type Prompts struct {
	Confirm func(local, remote identity.Fingerprint) bool
	Name    func(remote identity.Fingerprint) string
}

// Run pairs a provisional session. The local person must confirm before the
// confirmations are exchanged. Trust is stored only when both sides confirmed
// in this session. A declined comparison closes the session, so the other side
// sees the connection end instead of waiting.
func Run(ctx context.Context, s *transport.Session, id *identity.Identity, store *trust.Store, p Prompts) (string, error) {
	pr, err := s.Pairing()
	if err != nil {
		return "", err
	}
	remote, err := identity.FingerprintOf(s.PeerKey())
	if err != nil {
		return "", err
	}
	if !p.Confirm(id.Fingerprint(), remote) {
		_ = s.Close()
		return "", ErrCancelled
	}
	pr.ConfirmLocal()
	if err := transport.ExchangePairing(ctx, s, pr); err != nil {
		return "", err
	}
	name := ""
	if p.Name != nil {
		name = strings.TrimSpace(p.Name(remote))
	}
	if name == "" {
		name = remote.Hex()[:8]
	}
	if err := trust.CommitPairing(store, pr, name); err != nil {
		return "", err
	}
	if err := s.Promote(store); err != nil {
		return "", err
	}
	return name, nil
}
