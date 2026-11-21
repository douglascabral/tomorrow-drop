// Package apperr define os erros sentinela compartilhados pelo núcleo.
//
// Convenção: erros são envolvidos com fmt.Errorf("operação: %w", err) e
// verificados com errors.Is. Mensagens nunca contêm conteúdo transferido,
// chaves ou caminhos completos.
package apperr

import (
	"context"
	"errors"
)

var (
	ErrMalformed       = errors.New("mensagem malformada")
	ErrLimitExceeded   = errors.New("limite de recursos excedido")
	ErrTimeout         = errors.New("tempo limite excedido")
	ErrCanceled        = errors.New("operação cancelada")
	ErrUnauthorized    = errors.New("operação não autorizada")
	ErrIdentityChanged = errors.New("identidade alterada")
	ErrVersionMismatch = errors.New("versão de protocolo incompatível")
)

// FromContext converte erros de context.Context nos sentinelas do núcleo.
// Outros erros são devolvidos sem alteração.
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
