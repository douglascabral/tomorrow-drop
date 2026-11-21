package apperr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"syscall"
	"testing"
)

func TestFromContextMapsCancellation(t *testing.T) {
	if got := FromContext(context.Canceled); !errors.Is(got, ErrCanceled) {
		t.Fatalf("FromContext(Canceled) = %v, want ErrCanceled", got)
	}
}

func TestFromContextMapsTimeout(t *testing.T) {
	if got := FromContext(context.DeadlineExceeded); !errors.Is(got, ErrTimeout) {
		t.Fatalf("FromContext(DeadlineExceeded) = %v, want ErrTimeout", got)
	}
}

func TestFromContextPreservesWrappedError(t *testing.T) {
	wrapped := fmt.Errorf("send: %w", context.Canceled)
	if got := FromContext(wrapped); !errors.Is(got, ErrCanceled) {
		t.Fatalf("FromContext(wrapped) = %v, want ErrCanceled", got)
	}
}

func TestFromContextNilAndOtherErrorsAreUnchanged(t *testing.T) {
	if got := FromContext(nil); got != nil {
		t.Fatalf("FromContext(nil) = %v, want nil", got)
	}
	other := ErrMalformed
	if got := FromContext(other); !errors.Is(got, other) {
		t.Fatalf("FromContext(other) = %v, want the same error", got)
	}
}

func TestMessageKeyMapsEachSentinel(t *testing.T) {
	cases := map[error]string{
		ErrMalformed:       "error.malformed",
		ErrLimitExceeded:   "error.limit_exceeded",
		ErrTimeout:         "error.timeout",
		ErrCanceled:        "error.canceled",
		ErrUnauthorized:    "error.unauthorized",
		ErrIdentityChanged: "error.identity_changed",
		ErrVersionMismatch: "error.version_mismatch",
		ErrAlreadyTrusted:  "error.already_trusted",
		ErrNotTrusted:      "error.not_trusted",
	}
	for sentinel, want := range cases {
		if got := MessageKey(sentinel); got != want {
			t.Errorf("MessageKey(%v) = %q, want %q", sentinel, got, want)
		}
	}
}

func TestMessageKeyFindsSentinelInsideWrappedError(t *testing.T) {
	wrapped := fmt.Errorf("save: %w", fmt.Errorf("inner: %w", ErrUnauthorized))
	if got := MessageKey(wrapped); got != "error.unauthorized" {
		t.Fatalf("MessageKey(wrapped) = %q, want error.unauthorized", got)
	}
}

func TestMessageKeyUnknownAndNil(t *testing.T) {
	if got := MessageKey(errors.New("something else")); got != "error.unknown" {
		t.Fatalf("MessageKey(unknown) = %q, want error.unknown", got)
	}
	if got := MessageKey(nil); got != "" {
		t.Fatalf("MessageKey(nil) = %q, want empty", got)
	}
}

// Regression: a broken connection or a folder that cannot be written showed
// "An unexpected error occurred". They now have their own messages.
func TestMessageKeyNamesConnectionAndPermissionProblems(t *testing.T) {
	cases := map[string]error{
		"error.connection_lost": fmt.Errorf("read frame: %w", io.ErrUnexpectedEOF),
		"error.permission":      fmt.Errorf("create file: %w", fs.ErrPermission),
	}
	for want, err := range cases {
		if got := MessageKey(err); got != want {
			t.Errorf("MessageKey(%v) = %q, want %q", err, got, want)
		}
	}
}

// Regression: sending to a device that is off, or not on the network, showed
// "An unexpected error occurred". The person is told what to check.
func TestMessageKeyNamesUnreachableDevices(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}}
	cases := map[string]error{
		"error.unreachable": fmt.Errorf("dial: %w", refused),
		"error.no_route":    fmt.Errorf("dial: %w", syscall.EHOSTUNREACH),
		"error.timeout":     fmt.Errorf("dial: %w", os.ErrDeadlineExceeded),
	}
	for want, err := range cases {
		if got := MessageKey(err); got != want {
			t.Errorf("MessageKey(%v) = %q, want %q", err, got, want)
		}
	}
}
