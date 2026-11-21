// Package apperr defines the sentinel errors shared by the core.
//
// Convention: wrap errors with fmt.Errorf("operation: %w", err) and check them
// with errors.Is. Messages never contain transferred content, keys or full paths.
package apperr

import (
	"context"
	"errors"
)

var (
	ErrMalformed       = errors.New("malformed message")
	ErrLimitExceeded   = errors.New("resource limit exceeded")
	ErrTimeout         = errors.New("timeout exceeded")
	ErrCanceled        = errors.New("operation canceled")
	ErrUnauthorized    = errors.New("operation not authorized")
	ErrIdentityChanged = errors.New("identity changed")
	ErrVersionMismatch = errors.New("incompatible protocol version")
	ErrAlreadyTrusted  = errors.New("device already trusted")
	ErrNotTrusted      = errors.New("device not trusted")
)

// keys maps each sentinel to its translation key (see internal/i18n/locales).
var keys = []struct {
	err error
	key string
}{
	{ErrMalformed, "error.malformed"},
	{ErrLimitExceeded, "error.limit_exceeded"},
	{ErrTimeout, "error.timeout"},
	{ErrCanceled, "error.canceled"},
	{ErrUnauthorized, "error.unauthorized"},
	{ErrIdentityChanged, "error.identity_changed"},
	{ErrVersionMismatch, "error.version_mismatch"},
	{ErrAlreadyTrusted, "error.already_trusted"},
	{ErrNotTrusted, "error.not_trusted"},
}

// MessageKey returns the translation key for err, so interfaces can show a
// translated message instead of the internal English text. It returns
// "error.unknown" for errors without a sentinel, and "" for nil.
func MessageKey(err error) string {
	if err == nil {
		return ""
	}
	for _, k := range keys {
		if errors.Is(err, k.err) {
			return k.key
		}
	}
	return "error.unknown"
}

// FromContext maps context.Context errors to the core sentinels.
// Other errors are returned unchanged.
func FromContext(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return ErrCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrTimeout
	default:
		return err
	}
}
