// Package safelog cria loggers que redigem dados sensíveis.
package safelog

import (
	"io"
	"log/slog"
	"strings"
)

// Redacted substitui o valor de qualquer campo sensível.
const Redacted = "[REDACTED]"

// sensitiveKeys são fragmentos de nome de campo (sem distinção de maiúsculas)
// cujos valores nunca são gravados.
var sensitiveKeys = []string{
	"private",
	"secret",
	"token",
	"password",
	"key",
	"content",
	"clipboard",
	"payload",
	"filename",
	"path",
}

// New cria um slog.Logger que grava em w, redigindo campos sensíveis.
func New(w io.Writer, level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redact,
	}))
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindAny {
		if _, ok := a.Value.Any().([]byte); ok {
			return slog.String(a.Key, Redacted)
		}
	}
	if isSensitive(a.Key) {
		return slog.String(a.Key, Redacted)
	}
	return a
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, fragmento := range sensitiveKeys {
		if strings.Contains(lower, fragmento) {
			return true
		}
	}
	return false
}
