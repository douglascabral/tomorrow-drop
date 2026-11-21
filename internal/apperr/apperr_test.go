package apperr

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestFromContextMapeiaCancelamento(t *testing.T) {
	if got := FromContext(context.Canceled); !errors.Is(got, ErrCanceled) {
		t.Fatalf("FromContext(Canceled) = %v, esperado ErrCanceled", got)
	}
}

func TestFromContextMapeiaTimeout(t *testing.T) {
	if got := FromContext(context.DeadlineExceeded); !errors.Is(got, ErrTimeout) {
		t.Fatalf("FromContext(DeadlineExceeded) = %v, esperado ErrTimeout", got)
	}
}

func TestFromContextPreservaErroEmbrulhado(t *testing.T) {
	wrapped := fmt.Errorf("envio: %w", context.Canceled)
	if got := FromContext(wrapped); !errors.Is(got, ErrCanceled) {
		t.Fatalf("FromContext(envolvido) = %v, esperado ErrCanceled", got)
	}
}

func TestFromContextNilEOutrosErrosNaoSaoAlterados(t *testing.T) {
	if got := FromContext(nil); got != nil {
		t.Fatalf("FromContext(nil) = %v, esperado nil", got)
	}
	other := ErrMalformed
	if got := FromContext(other); !errors.Is(got, other) {
		t.Fatalf("FromContext(outro) = %v, esperado o mesmo erro", got)
	}
}
