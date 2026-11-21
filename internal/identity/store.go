package identity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/storage"
)

// FileName is the name of the file holding the private key inside Store.Dir.
const FileName = "identity.key"

// Store persists the local identity on disk with restrictive permissions.
//
// Create and Load use the os package errors: ErrNotExist when the file does
// not exist and ErrExist when it already does. Corrupted files result in
// apperr.ErrMalformed and are never silently overwritten.
type Store struct {
	Dir string
}

// Load reads and validates the existing identity.
func (s Store) Load() (*Identity, error) {
	data, err := os.ReadFile(s.path()) //nolint:gosec // G304: identity file in the user's own configuration directory
	if err != nil {
		return nil, fmt.Errorf("read identity: %w", err)
	}
	return decode(data)
}

// Create generates a new identity only if none exists.
func (s Store) Create() (*Identity, error) {
	id, err := Generate()
	if err != nil {
		return nil, err
	}
	data, err := encode(id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create identity directory: %w", err)
	}
	if err := storage.WriteFileExclusive(s.path(), data, 0o600); err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}
	return id, nil
}

// Replace generates a new identity and replaces the previous one. It must be
// called only after explicit user confirmation, because the old identity
// stops existing.
func (s Store) Replace() (*Identity, error) {
	id, err := Generate()
	if err != nil {
		return nil, err
	}
	data, err := encode(id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create identity directory: %w", err)
	}
	if err := storage.WriteFileAtomic(s.path(), data, 0o600); err != nil {
		return nil, fmt.Errorf("replace identity: %w", err)
	}
	return id, nil
}

// LoadOrCreate reuses the existing identity or creates one if it is missing.
func (s Store) LoadOrCreate() (*Identity, error) {
	id, err := s.Load()
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	id, err = s.Create()
	if errors.Is(err, os.ErrExist) {
		// Another instance created the identity between the two calls.
		return s.Load()
	}
	return id, err
}

func (s Store) path() string {
	return filepath.Join(s.Dir, FileName)
}

func encode(id *Identity) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(id.priv)
	if err != nil {
		return nil, fmt.Errorf("encode private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func decode(data []byte) (*Identity, error) {
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, fmt.Errorf("invalid identity file: %w", apperr.ErrMalformed)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperr.ErrMalformed, err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok || len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key is not Ed25519: %w", apperr.ErrMalformed)
	}
	return &Identity{priv: priv}, nil
}
