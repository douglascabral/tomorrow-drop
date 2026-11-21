// Package trust stores trusted public keys and drives the pairing flow (PRD §9).
package trust

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/storage"
)

const (
	fileVersion  = 1
	maxNameRunes = 64
)

var (
	// ErrAlreadyTrusted means the key is already in the trust list.
	ErrAlreadyTrusted = apperr.ErrAlreadyTrusted
	// ErrNotTrusted means the key is not in the trust list.
	ErrNotTrusted = apperr.ErrNotTrusted
)

// Peer is a trusted device.
type Peer struct {
	Name        string
	Key         ed25519.PublicKey
	Fingerprint identity.Fingerprint
	AddedAt     time.Time
}

// Store persists the list of trusted devices. Trust enters the list only
// through CommitPairing, after proven mutual confirmation.
//
// Read failures make IsTrusted return false (fail closed).
type Store struct {
	path string
	mu   sync.Mutex
}

// Open binds a Store to the given file. The file is created on the first write.
func Open(path string) *Store {
	return &Store{path: path}
}

// IsTrusted reports whether the public key is in the trust list.
func (s *Store) IsTrusted(pub ed25519.PublicKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	peers, err := s.load()
	if err != nil {
		return false
	}
	for _, p := range peers {
		if bytes.Equal(p.Key, pub) {
			return true
		}
	}
	return false
}

// List returns the trusted devices.
func (s *Store) List() ([]Peer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Remove revokes trust in a device. A new connection will require new pairing.
func (s *Store) Remove(fp identity.Fingerprint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	peers, err := s.load()
	if err != nil {
		return err
	}
	var remaining []Peer
	found := false
	for _, p := range peers {
		if p.Fingerprint == fp {
			found = true
			continue
		}
		remaining = append(remaining, p)
	}
	if !found {
		return ErrNotTrusted
	}
	return s.save(remaining)
}

// Rename changes the friendly name of a trusted device.
func (s *Store) Rename(fp identity.Fingerprint, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	peers, err := s.load()
	if err != nil {
		return err
	}
	for i := range peers {
		if peers[i].Fingerprint == fp {
			peers[i].Name = name
			return s.save(peers)
		}
	}
	return ErrNotTrusted
}

// add records a trusted key. It is called only by CommitPairing.
func (s *Store) add(pub ed25519.PublicKey, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	fp, err := identity.FingerprintOf(pub)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	peers, err := s.load()
	if err != nil {
		return err
	}
	for _, p := range peers {
		if p.Fingerprint == fp {
			return ErrAlreadyTrusted
		}
	}
	peers = append(peers, Peer{
		Name:        name,
		Key:         append(ed25519.PublicKey(nil), pub...),
		Fingerprint: fp,
		AddedAt:     time.Now().UTC(),
	})
	return s.save(peers)
}

// ValidateName accepts names of 1 to 64 characters, without control
// characters and not made only of spaces.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("empty name: %w", apperr.ErrMalformed)
	}
	if utf8.RuneCountInString(name) > maxNameRunes {
		return fmt.Errorf("name longer than %d characters: %w", maxNameRunes, apperr.ErrMalformed)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("name contains control characters: %w", apperr.ErrMalformed)
		}
	}
	return nil
}

type record struct {
	Name        string    `json:"name"`
	PublicKey   []byte    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	AddedAt     time.Time `json:"added_at"`
}

type fileFormat struct {
	Version int      `json:"version"`
	Peers   []record `json:"peers"`
}

func (s *Store) load() ([]Peer, error) {
	data, err := os.ReadFile(s.path) //nolint:gosec // G304: trust file in the user's own configuration directory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read trust list: %w", err)
	}
	var f fileFormat
	if err := json.Unmarshal(data, &f); err != nil || f.Version != fileVersion {
		return nil, fmt.Errorf("invalid trust list file: %w", apperr.ErrMalformed)
	}
	peers := make([]Peer, 0, len(f.Peers))
	for _, r := range f.Peers {
		p, err := r.peer()
		if err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, nil
}

func (s *Store) save(peers []Peer) error {
	f := fileFormat{Version: fileVersion, Peers: make([]record, 0, len(peers))}
	for _, p := range peers {
		f.Peers = append(f.Peers, record{
			Name:        p.Name,
			PublicKey:   p.Key,
			Fingerprint: p.Fingerprint.Hex(),
			AddedAt:     p.AddedAt,
		})
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode trust list: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create trust list directory: %w", err)
	}
	return storage.WriteFileAtomic(s.path, data, 0o600)
}

func (r record) peer() (Peer, error) {
	fp, err := identity.FingerprintOf(r.PublicKey)
	if err != nil {
		return Peer{}, err
	}
	if fp.Hex() != r.Fingerprint {
		return Peer{}, fmt.Errorf("fingerprint does not match the key: %w", apperr.ErrMalformed)
	}
	return Peer{
		Name:        r.Name,
		Key:         ed25519.PublicKey(r.PublicKey),
		Fingerprint: fp,
		AddedAt:     r.AddedAt,
	}, nil
}
