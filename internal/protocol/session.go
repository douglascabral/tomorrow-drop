package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// SessionID identifica uma sessão de conexão. Não é uma credencial de
// autenticação: a identidade é provada pelo TLS e pela chave Ed25519.
type SessionID [16]byte

// NewSessionID gera um identificador aleatório com crypto/rand.
func NewSessionID() (SessionID, error) {
	var id SessionID
	if _, err := rand.Read(id[:]); err != nil {
		return SessionID{}, fmt.Errorf("gerar sessão: %w", err)
	}
	return id, nil
}

// String devolve o identificador em hexadecimal (32 caracteres).
func (id SessionID) String() string {
	return hex.EncodeToString(id[:])
}
