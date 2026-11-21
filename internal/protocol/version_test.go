package protocol

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func TestNegotiateChoosesHighestCommonVersion(t *testing.T) {
	got, err := Negotiate([]Version{1, 2}, []Version{3, 2, 1})
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if got != 2 {
		t.Fatalf("Negotiate = %d, want 2", got)
	}
}

func TestNegotiateWithoutCommonVersionFails(t *testing.T) {
	_, err := Negotiate([]Version{1}, []Version{2})
	if !errors.Is(err, apperr.ErrVersionMismatch) {
		t.Fatalf("Negotiate without common version = %v, want ErrVersionMismatch", err)
	}
}

func TestNegotiateEmptyRemoteListFails(t *testing.T) {
	_, err := Negotiate([]Version{1}, nil)
	if !errors.Is(err, apperr.ErrVersionMismatch) {
		t.Fatalf("Negotiate with empty list = %v, want ErrVersionMismatch", err)
	}
}

func TestCurrentVersionIsSupported(t *testing.T) {
	if _, err := Negotiate(SupportedVersions, []Version{CurrentVersion}); err != nil {
		t.Fatalf("current version does not negotiate with itself: %v", err)
	}
}
