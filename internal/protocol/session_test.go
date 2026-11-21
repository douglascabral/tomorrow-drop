package protocol

import (
	"encoding/hex"
	"testing"
)

func TestNewSessionIDTem128Bits(t *testing.T) {
	id, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if len(id) != 16 {
		t.Fatalf("len(SessionID) = %d, esperado 16", len(id))
	}
	if len(id.String()) != 32 {
		t.Fatalf("String() = %q, esperado 32 caracteres hexadecimais", id.String())
	}
	if _, err := hex.DecodeString(id.String()); err != nil {
		t.Fatalf("String() não é hexadecimal: %v", err)
	}
}

func TestNewSessionIDNaoRepete(t *testing.T) {
	primeiro, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	segundo, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if primeiro == segundo {
		t.Fatalf("duas sessões geradas são iguais: %s", primeiro)
	}
}
