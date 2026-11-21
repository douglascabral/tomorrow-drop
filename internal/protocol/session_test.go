package protocol

import (
	"encoding/hex"
	"testing"
)

func TestNewSessionIDHas128Bits(t *testing.T) {
	id, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if len(id) != 16 {
		t.Fatalf("len(SessionID) = %d, want 16", len(id))
	}
	if len(id.String()) != 32 {
		t.Fatalf("String() = %q, want 32 hexadecimal characters", id.String())
	}
	if _, err := hex.DecodeString(id.String()); err != nil {
		t.Fatalf("String() is not hexadecimal: %v", err)
	}
}

func TestNewSessionIDDoesNotRepeat(t *testing.T) {
	first, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	second, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if first == second {
		t.Fatalf("two generated sessions are equal: %s", first)
	}
}
