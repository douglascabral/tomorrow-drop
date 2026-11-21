package transfer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
)

func TestPublishExclusiveNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(t.TempDir(), "tdrop-1.part")
	if err := os.WriteFile(staged, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	final, err := publishExclusive(staged, dir, "a.txt")
	if err != nil {
		t.Fatalf("publishExclusive: %v", err)
	}
	if filepath.Base(final) != "a (1).txt" {
		t.Fatalf("final name = %q, want %q", filepath.Base(final), "a (1).txt")
	}
	original, _ := os.ReadFile(filepath.Join(dir, "a.txt")) //nolint:gosec // G304: path inside t.TempDir()
	if string(original) != "original" {
		t.Fatalf("existing file was overwritten")
	}
	if _, err := os.Stat(staged); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged file was not removed after publishing")
	}
}

func TestPublishExclusiveUsesPlainNameWhenFree(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(t.TempDir(), "tdrop-2.part")
	_ = os.WriteFile(staged, []byte("x"), 0o600)
	final, err := publishExclusive(staged, dir, "free.txt")
	if err != nil {
		t.Fatalf("publishExclusive: %v", err)
	}
	if filepath.Base(final) != "free.txt" {
		t.Fatalf("final name = %q, want free.txt", filepath.Base(final))
	}
}

func TestCleanupAbandonedRemovesOnlyOldStagingFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	old := filepath.Join(dir, "tdrop-old.part")
	fresh := filepath.Join(dir, "tdrop-fresh.part")
	foreign := filepath.Join(dir, "important-document.txt")
	for _, p := range []string{old, fresh, foreign} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	past := now.Add(-2 * time.Hour)
	for _, p := range []string{old, foreign} {
		if err := os.Chtimes(p, past, past); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := CleanupAbandoned(dir, time.Hour, now)
	if err != nil {
		t.Fatalf("CleanupAbandoned: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(old); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old staging file was not removed")
	}
	for _, keep := range []string{fresh, foreign} {
		if _, err := os.Stat(keep); err != nil {
			t.Errorf("%s was removed but must be kept: %v", filepath.Base(keep), err)
		}
	}
}

func TestDesktopDestinationRejectsMissingDirectory(t *testing.T) {
	d := DesktopDestination{Dir: filepath.Join(t.TempDir(), "missing")}
	staged := filepath.Join(t.TempDir(), "tdrop-3.part")
	_ = os.WriteFile(staged, []byte("x"), 0o600)
	if _, err := d.Publish(staged, "a.txt"); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("Publish into missing dir = %v, want ErrMalformed", err)
	}
}

func TestDefaultDownloadsDirIsUnderHome(t *testing.T) {
	dir, err := DefaultDownloadsDir()
	if err != nil {
		t.Fatalf("DefaultDownloadsDir: %v", err)
	}
	if filepath.Base(dir) != "Downloads" {
		t.Fatalf("DefaultDownloadsDir = %q, want a Downloads folder", dir)
	}
}
