// Package transport opens TLS 1.3 connections authenticated by Ed25519
// identities (PRD §8). An established connection is not trusted by itself:
// the security state is derived from the trust list after the handshake.
package transport

import (
	"bytes"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
)

const (
	// certValidity is the lifetime of the session certificate. The identity
	// does not change, so the certificate may be renewed on every run.
	certValidity = 365 * 24 * time.Hour
	// bindingLabel identifies the TLS exporter used by pairing.
	bindingLabel = "tomorrow-drop pairing v1"
	bindingSize  = 32
)

// TrustLookup reports whether a public key is trusted. Implemented by trust.Store.
type TrustLookup interface {
	IsTrusted(pub ed25519.PublicKey) bool
}

func certificateFor(id *identity.Identity) (tls.Certificate, error) {
	now := time.Now()
	return identity.NewCertificate(id, now.Add(-time.Hour), now.Add(certValidity))
}

// tlsConfig requires TLS 1.3 and a certificate from both sides. If pin is not
// nil, only that key is accepted.
func tlsConfig(cert tls.Certificate, pin ed25519.PublicKey) *tls.Config {
	// nosemgrep: problem-based-packs.insecure-transport.go-stdlib.bypass-tls-verification.bypass-tls-verification
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAnyClientCert,
		// Without tickets, every connection performs a full handshake and goes through verifyPeer.
		SessionTicketsDisabled: true,
		// Go's default chain validation does not apply to self-signed Ed25519
		// certificates. The real validation is VerifyPeerCertificate, which runs on every handshake.
		InsecureSkipVerify:    true, //nolint:gosec // G402: default validation replaced by verifyPeer
		VerifyPeerCertificate: verifyPeer(pin),
	}
}

func verifyPeer(pin ed25519.PublicKey) func([][]byte, [][]*x509.Certificate) error {
	return func(raw [][]byte, _ [][]*x509.Certificate) error {
		if len(raw) != 1 {
			return fmt.Errorf("unexpected certificate chain: %w", apperr.ErrMalformed)
		}
		key, err := identity.VerifyCertificate(raw[0], time.Now())
		if err != nil {
			return err
		}
		if pin != nil && !bytes.Equal(key, pin) {
			return fmt.Errorf("peer key differs from the pinned key: %w", apperr.ErrIdentityChanged)
		}
		return nil
	}
}
