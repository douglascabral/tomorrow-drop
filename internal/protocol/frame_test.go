package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

func TestQuadroIdaEVolta(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte(`{"type":"DISCOVER"}`)

	if err := WriteFrame(&buf, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("ReadFrame = %q, esperado %q", got, payload)
	}
}

func TestQuadroUsaCabecalhoBigEndian(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, []byte("abc")); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	if got := binary.BigEndian.Uint32(buf.Bytes()[:4]); got != 3 {
		t.Fatalf("cabeçalho = %d, esperado 3", got)
	}
}

func TestWriteFrameRejeitaPayloadVazio(t *testing.T) {
	err := WriteFrame(io.Discard, nil)
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("WriteFrame vazio = %v, esperado ErrMalformed", err)
	}
}

func TestWriteFrameRejeitaPayloadAcimaDoLimite(t *testing.T) {
	err := WriteFrame(io.Discard, make([]byte, limits.MaxControlFrameBytes+1))
	if !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Fatalf("WriteFrame grande = %v, esperado ErrLimitExceeded", err)
	}
}

func TestReadFrameRejeitaTamanhoAcimaDoLimiteSemLerCorpo(t *testing.T) {
	var cabecalho [4]byte
	binary.BigEndian.PutUint32(cabecalho[:], limits.MaxControlFrameBytes+1)
	leitor := bytes.NewReader(cabecalho[:])

	_, err := ReadFrame(leitor)
	if !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Fatalf("ReadFrame = %v, esperado ErrLimitExceeded", err)
	}
	if leitor.Len() != 0 {
		t.Fatalf("ReadFrame leu o corpo de um quadro rejeitado")
	}
}

func TestReadFrameRejeitaTamanhoZero(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader([]byte{0, 0, 0, 0}))
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("ReadFrame zero = %v, esperado ErrMalformed", err)
	}
}

func TestReadFrameRejeitaCorpoTruncado(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader([]byte{0, 0, 0, 5, 'a', 'b'}))
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("ReadFrame truncado = %v, esperado ErrMalformed", err)
	}
}

func TestReadFrameRejeitaCabecalhoIncompleto(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader([]byte{0, 0}))
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("ReadFrame cabeçalho parcial = %v, esperado ErrMalformed", err)
	}
}

func TestReadFrameFimLimpoRetornaEOF(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("ReadFrame vazio = %v, esperado io.EOF", err)
	}
}
