package safelog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func logOutput(t *testing.T, emit func(*slog.Logger)) string {
	t.Helper()
	var buf bytes.Buffer
	emit(New(&buf, slog.LevelDebug))
	return buf.String()
}

func TestRedactsPrivateKeys(t *testing.T) {
	out := logOutput(t, func(l *slog.Logger) {
		l.Info("identity", slog.String("private_key", "SECRET-123"))
	})
	if strings.Contains(out, "SECRET-123") {
		t.Fatalf("private key appeared in the log: %s", out)
	}
	if !strings.Contains(out, Redacted) {
		t.Fatalf("expected marker %q in the log: %s", Redacted, out)
	}
}

func TestRedactsContentAndTokens(t *testing.T) {
	out := logOutput(t, func(l *slog.Logger) {
		l.Info("send",
			slog.String("clipboard", "private-text"),
			slog.String("token", "abc"),
			slog.String("filename", "contract.pdf"),
		)
	})
	for _, value := range []string{"private-text", "abc", "contract.pdf"} {
		if strings.Contains(out, value) {
			t.Fatalf("sensitive value %q appeared in the log: %s", value, out)
		}
	}
}

func TestRedactsRawBytes(t *testing.T) {
	out := logOutput(t, func(l *slog.Logger) {
		l.Info("frame", slog.Any("data", []byte("raw-content")))
	})
	if strings.Contains(out, "raw-content") {
		t.Fatalf("raw bytes appeared in the log: %s", out)
	}
}

func TestDoesNotRedactContextKeyBySubstring(t *testing.T) {
	out := logOutput(t, func(l *slog.Logger) {
		l.Info("error", slog.String("context", "handshake"))
	})
	if !strings.Contains(out, "handshake") {
		t.Fatalf("context field was redacted incorrectly: %s", out)
	}
}

func TestKeepsNonSensitiveFields(t *testing.T) {
	out := logOutput(t, func(l *slog.Logger) {
		l.Info("discovery", slog.String("device", "living-room-laptop"), slog.String("fingerprint", "ab12"))
	})
	for _, value := range []string{"living-room-laptop", "ab12"} {
		if !strings.Contains(out, value) {
			t.Fatalf("non-sensitive field %q was removed: %s", value, out)
		}
	}
}
