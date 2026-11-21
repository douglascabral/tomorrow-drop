package protocol

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func TestNegotiateEscolheMaiorVersaoComum(t *testing.T) {
	got, err := Negotiate([]Version{1, 2}, []Version{3, 2, 1})
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if got != 2 {
		t.Fatalf("Negotiate = %d, esperado 2", got)
	}
}

func TestNegotiateSemVersaoComumFalha(t *testing.T) {
	_, err := Negotiate([]Version{1}, []Version{2})
	if !errors.Is(err, apperr.ErrVersionMismatch) {
		t.Fatalf("Negotiate sem versão comum = %v, esperado ErrVersionMismatch", err)
	}
}

func TestNegotiateListaRemotaVaziaFalha(t *testing.T) {
	_, err := Negotiate([]Version{1}, nil)
	if !errors.Is(err, apperr.ErrVersionMismatch) {
		t.Fatalf("Negotiate com lista vazia = %v, esperado ErrVersionMismatch", err)
	}
}

func TestVersaoAtualEstaNaListaSuportada(t *testing.T) {
	if _, err := Negotiate(SupportedVersions, []Version{CurrentVersion}); err != nil {
		t.Fatalf("versão atual não negocia consigo mesma: %v", err)
	}
}
