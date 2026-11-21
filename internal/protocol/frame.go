// Package protocol define o framing, a versão, os estados e os limites
// do protocolo de transferência. Ver docs/PROTOCOLO.md.
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

const frameHeaderSize = 4

// WriteFrame grava payload precedido de seu tamanho (uint32, big-endian).
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("quadro vazio: %w", apperr.ErrMalformed)
	}
	if len(payload) > limits.MaxControlFrameBytes {
		return fmt.Errorf("quadro de %d bytes: %w", len(payload), apperr.ErrLimitExceeded)
	}

	var cabecalho [frameHeaderSize]byte
	binary.BigEndian.PutUint32(cabecalho[:], uint32(len(payload))) //nolint:gosec // G115: len(payload) já limitado a MaxControlFrameBytes acima.
	if _, err := w.Write(cabecalho[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadFrame lê um quadro. Tamanhos inválidos são rejeitados antes de
// qualquer alocação do corpo. io.EOF é devolvido somente se a conexão
// for encerrada antes de qualquer byte do cabeçalho.
func ReadFrame(r io.Reader) ([]byte, error) {
	var cabecalho [frameHeaderSize]byte
	if _, err := io.ReadFull(r, cabecalho[:]); err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("cabeçalho incompleto: %w", apperr.ErrMalformed)
	}

	tamanho := binary.BigEndian.Uint32(cabecalho[:])
	if tamanho == 0 {
		return nil, fmt.Errorf("quadro vazio: %w", apperr.ErrMalformed)
	}
	if tamanho > limits.MaxControlFrameBytes {
		return nil, fmt.Errorf("quadro de %d bytes: %w", tamanho, apperr.ErrLimitExceeded)
	}

	payload := make([]byte, tamanho)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("corpo truncado: %w", apperr.ErrMalformed)
	}
	return payload, nil
}
