package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tomorrow-drop/internal/i18n"
)

func portugueseEnv(t *testing.T) env {
	t.Helper()
	tr, err := i18n.Load("pt-BR")
	if err != nil {
		t.Fatalf("Load pt-BR: %v", err)
	}
	return env{
		Tr:         tr,
		ConfigPath: filepath.Join(t.TempDir(), "settings.json"),
	}
}

func TestVersionPrintsVersionAndProtocol(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb, portugueseEnv(t)); code != 0 {
		t.Fatalf("run(version) = %d, want 0 (stderr: %s)", code, errb.String())
	}
	if !strings.Contains(out.String(), "protocolo 1") {
		t.Fatalf("output = %q, want the protocol version", out.String())
	}
}

func TestNoArgumentsReturnsUsageWithErrorCode(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, &out, &errb, portugueseEnv(t)); code != 2 {
		t.Fatalf("run() = %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "uso:") {
		t.Fatalf("stderr = %q, want the usage message in pt-BR", errb.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout should be empty on error, got %q", out.String())
	}
}

func TestUnknownCommandReturnsErrorCode(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"unknown"}, &out, &errb, portugueseEnv(t)); code != 2 {
		t.Fatalf("run(unknown) = %d, want 2", code)
	}
}

func TestConfigLanguageSavesAndCanBeQueried(t *testing.T) {
	e := portugueseEnv(t)
	var out, errb bytes.Buffer
	if code := run([]string{"config", "language", "en"}, &out, &errb, e); code != 0 {
		t.Fatalf("config language en = %d (stderr: %s)", code, errb.String())
	}
	out.Reset()
	if code := run([]string{"config", "language"}, &out, &errb, e); code != 0 {
		t.Fatalf("config language = %d", code)
	}
	if !strings.Contains(out.String(), "en") {
		t.Fatalf("saved language did not appear in the query: %q", out.String())
	}
}

func TestConfigLanguageRejectsInvalidValue(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"config", "language", "klingon"}, &out, &errb, portugueseEnv(t)); code != 2 {
		t.Fatalf("invalid language = %d, want 2", code)
	}
	if errb.Len() == 0 {
		t.Fatalf("invalid language error was not reported")
	}
}

func TestConfigLanguageShowsTranslatedErrorForCorruptedSettings(t *testing.T) {
	e := portugueseEnv(t)
	if err := os.WriteFile(e.ConfigPath, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := run([]string{"config", "language"}, &out, &errb, e); code != 1 {
		t.Fatalf("corrupted settings = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "Dados malformados.") {
		t.Fatalf("stderr = %q, want the translated pt-BR error", errb.String())
	}
	if strings.Contains(errb.String(), "invalid settings") || strings.Contains(errb.String(), "invalid configuration") {
		t.Fatalf("stderr leaked the untranslated internal message: %q", errb.String())
	}
}
