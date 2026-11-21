package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func testPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "cfg", "settings.json")
}

func TestLoadWithoutFileUsesAuto(t *testing.T) {
	cfg, err := Load(testPath(t))
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if cfg.Language != "auto" {
		t.Fatalf("default language = %q, want auto", cfg.Language)
	}
}

func TestSaveAndReload(t *testing.T) {
	path := testPath(t)
	if err := Save(path, Settings{Language: "en"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Language != "en" {
		t.Fatalf("reloaded language = %q, want en", cfg.Language)
	}
}

func TestSaveRejectsUnsupportedLanguage(t *testing.T) {
	err := Save(testPath(t), Settings{Language: "klingon"})
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("Save klingon = %v, want ErrMalformed", err)
	}
}

func TestLoadRejectsCorruptedFile(t *testing.T) {
	path := testPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("Load corrupted = %v, want ErrMalformed", err)
	}
}
