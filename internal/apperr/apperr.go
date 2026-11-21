// Package apperr defines the sentinel errors shared by the core.
//
// Convention: wrap errors with fmt.Errorf("operation: %w", err) and check them
// with errors.Is. Messages never contain transferred content, keys or full paths.
package apperr

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"syscall"
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
	ErrNoRoute         = errors.New("no network route")
	ErrRateLimited     = errors.New("too many requests")
	ErrDeclined        = errors.New("transfer declined")
	ErrIntegrity       = errors.New("integrity check failed")
	ErrBusy            = errors.New("receiver busy")
	// ErrNoOffer means a connection ended before any transfer started. It is not
	// a failed transfer, so interfaces do not report it.
	ErrNoOffer = errors.New("no transfer offer received")
	// ErrPeerWantsToPair means a device the receiver still has paired asks to pair
	// again: it unpaired this device on its side.
	ErrPeerWantsToPair = errors.New("paired device asks to pair again")
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
	{ErrNoRoute, "error.no_route"},
	{ErrRateLimited, "error.rate_limited"},
	{ErrDeclined, "error.declined"},
	{ErrIntegrity, "error.integrity"},
	{ErrBusy, "error.busy"},
	{ErrNoOffer, "error.no_offer"},
	{ErrPeerWantsToPair, "error.peer_wants_to_pair"},
	// Standard errors that mean a broken connection or a folder that cannot be
	// written. They are named so the person knows what to check.
	{io.ErrUnexpectedEOF, "error.connection_lost"},
	{io.EOF, "error.connection_lost"},
	{fs.ErrPermission, "error.permission"},
	// A device that is off, not running the app or off the network.
	{syscall.ECONNREFUSED, "error.unreachable"},
	{syscall.EHOSTUNREACH, "error.no_route"},
	{syscall.ENETUNREACH, "error.no_route"},
	// Windows reports the same cases with Winsock codes that Go does not export
	// as syscall constants: 10061 refused, 10051 network unreachable, 10065 host
	// unreachable. The numbers are fixed by the Windows API.
	{syscall.Errno(10061), "error.unreachable"},
	{syscall.Errno(10051), "error.no_route"},
	{syscall.Errno(10065), "error.no_route"},
	{os.ErrDeadlineExceeded, "error.timeout"},
	{context.DeadlineExceeded, "error.timeout"},
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
