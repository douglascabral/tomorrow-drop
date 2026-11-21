package safelog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func registrar(t *testing.T, emitir func(*slog.Logger)) string {
	t.Helper()
	var buf bytes.Buffer
	emitir(New(&buf, slog.LevelDebug))
	return buf.String()
}

func TestRedigeChavesPrivadas(t *testing.T) {
	out := registrar(t, func(l *slog.Logger) {
		l.Info("identidade", slog.String("private_key", "SEGREDO-123"))
	})
	if strings.Contains(out, "SEGREDO-123") {
		t.Fatalf("chave privada apareceu no log: %s", out)
	}
	if !strings.Contains(out, Redacted) {
		t.Fatalf("esperado marcador %q no log: %s", Redacted, out)
	}
}

func TestRedigeConteudoETokens(t *testing.T) {
	out := registrar(t, func(l *slog.Logger) {
		l.Info("envio",
			slog.String("clipboard", "texto-privado"),
			slog.String("token", "abc"),
			slog.String("filename", "contrato.pdf"),
		)
	})
	for _, valor := range []string{"texto-privado", "abc", "contrato.pdf"} {
		if strings.Contains(out, valor) {
			t.Fatalf("valor sensível %q apareceu no log: %s", valor, out)
		}
	}
}

func TestRedigeBytesBrutos(t *testing.T) {
	out := registrar(t, func(l *slog.Logger) {
		l.Info("frame", slog.Any("data", []byte("conteudo-bruto")))
	})
	if strings.Contains(out, "conteudo-bruto") {
		t.Fatalf("bytes brutos apareceram no log: %s", out)
	}
}

func TestNaoRedigeChaveContextPorSubstring(t *testing.T) {
	out := registrar(t, func(l *slog.Logger) {
		l.Info("erro", slog.String("context", "handshake"))
	})
	if !strings.Contains(out, "handshake") {
		t.Fatalf("campo context foi redigido indevidamente: %s", out)
	}
}

func TestMantemCamposNaoSensiveis(t *testing.T) {
	out := registrar(t, func(l *slog.Logger) {
		l.Info("descoberta", slog.String("device", "laptop-sala"), slog.String("fingerprint", "ab12"))
	})
	for _, valor := range []string{"laptop-sala", "ab12"} {
		if !strings.Contains(out, valor) {
			t.Fatalf("campo não sensível %q foi removido: %s", valor, out)
		}
	}
}
