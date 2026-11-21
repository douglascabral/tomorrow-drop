package transfer

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

// Kind is the type of content in an offer.
type Kind string

const (
	KindFile Kind = "file"
	KindText Kind = "text"
	KindLink Kind = "link"
)

// Item describes one file or the single text or link of an offer.
type Item struct {
	Name string
	Size int64
}

// Offer is what the recipient sees before deciding. Peer is the authenticated
// public key of the sender.
type Offer struct {
	ID    string
	Kind  Kind
	Items []Item
	Peer  ed25519.PublicKey
}

// ValidateName accepts only a plain file name: no path separators, no drive
// or stream syntax, no control characters, no hidden or relative names.
func ValidateName(name string) error {
	switch {
	case name == "", name == ".", name == "..":
		return fmt.Errorf("file name %q: %w", name, apperr.ErrMalformed)
	case strings.HasPrefix(name, "."):
		return fmt.Errorf("hidden file name %q: %w", name, apperr.ErrMalformed)
	case len(name) > limits.MaxFileNameBytes:
		return fmt.Errorf("file name longer than %d bytes: %w", limits.MaxFileNameBytes, apperr.ErrMalformed)
	case strings.ContainsAny(name, `/\:`):
		return fmt.Errorf("file name %q contains a path or stream separator: %w", name, apperr.ErrMalformed)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("file name contains control characters: %w", apperr.ErrMalformed)
		}
	}
	return nil
}

// ValidateOffer checks the offer before any file is created.
func ValidateOffer(o Offer) error {
	if len(o.ID) != 16 {
		return fmt.Errorf("offer ID: %w", apperr.ErrMalformed)
	}
	if _, err := hex.DecodeString(o.ID); err != nil {
		return fmt.Errorf("offer ID is not hexadecimal: %w", apperr.ErrMalformed)
	}
	switch o.Kind {
	case KindFile:
		return validateFiles(o.Items)
	case KindText, KindLink:
		if len(o.Items) != 1 {
			return fmt.Errorf("%s offer must have exactly one item: %w", o.Kind, apperr.ErrMalformed)
		}
		it := o.Items[0]
		if it.Name != "" {
			return fmt.Errorf("%s offer must not carry a file name: %w", o.Kind, apperr.ErrMalformed)
		}
		if it.Size < 0 {
			return fmt.Errorf("negative size: %w", apperr.ErrMalformed)
		}
		if it.Size > limits.MaxTextBytes {
			return fmt.Errorf("%s of %d bytes: %w", o.Kind, it.Size, apperr.ErrLimitExceeded)
		}
		return nil
	default:
		return fmt.Errorf("offer kind %q: %w", o.Kind, apperr.ErrMalformed)
	}
}

func validateFiles(items []Item) error {
	if len(items) == 0 {
		return fmt.Errorf("offer without files: %w", apperr.ErrMalformed)
	}
	if len(items) > limits.MaxFilesPerOffer {
		return fmt.Errorf("%d files in one offer: %w", len(items), apperr.ErrLimitExceeded)
	}
	var total int64
	for _, it := range items {
		if err := ValidateName(it.Name); err != nil {
			return err
		}
		if it.Size < 0 {
			return fmt.Errorf("negative size: %w", apperr.ErrMalformed)
		}
		if it.Size > limits.MaxFileBytes {
			return fmt.Errorf("file of %d bytes: %w", it.Size, apperr.ErrLimitExceeded)
		}
		total += it.Size
		if total > limits.MaxOfferBytes {
			return fmt.Errorf("offer above %d bytes: %w", int64(limits.MaxOfferBytes), apperr.ErrLimitExceeded)
		}
	}
	return nil
}

// ValidateLink accepts only absolute http and https URLs. Other schemes such as
// file, javascript or mailto are refused.
func ValidateLink(link string) error {
	if link == "" || !utf8.ValidString(link) {
		return fmt.Errorf("link: %w", apperr.ErrMalformed)
	}
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return fmt.Errorf("link is not an absolute URL: %w", apperr.ErrMalformed)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("link scheme %q is not allowed: %w", u.Scheme, apperr.ErrMalformed)
	}
	return nil
}

func newOfferID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b[:])
}
