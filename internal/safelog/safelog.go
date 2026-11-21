// Package safelog creates loggers that redact sensitive data.
package safelog

import (
	"io"
	"log/slog"
	"strings"
)

// Redacted replaces the value of any sensitive field.
const Redacted = "[REDACTED]"

// sensitiveKeys are field name fragments (case-insensitive) whose values
// are never written.
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

// New creates a slog.Logger that writes to w, redacting sensitive fields.
func New(w io.Writer, level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redact,
	}))
}

func redact(_ []string, attr slog.Attr) slog.Attr {
	if attr.Value.Kind() == slog.KindAny {
		if _, ok := attr.Value.Any().([]byte); ok {
			return slog.String(attr.Key, Redacted)
		}
	}
	if isSensitive(attr.Key) {
		return slog.String(attr.Key, Redacted)
	}
	return attr
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, fragment := range sensitiveKeys {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}
