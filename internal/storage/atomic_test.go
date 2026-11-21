package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileExclusiveCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := WriteFileExclusive(path, []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFileExclusive: %v", err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: path inside t.TempDir() of this test
	if err != nil || string(data) != "content" {
		t.Fatalf("content = %q (%v), want %q", data, err, "content")
	}
}

func TestWriteFileExclusiveDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := WriteFileExclusive(path, []byte("new"), 0o600)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("WriteFileExclusive on existing file = %v, want os.ErrExist", err)
	}
	data, _ := os.ReadFile(path) //nolint:gosec // G304: path inside t.TempDir() of this test
	if string(data) != "original" {
		t.Fatalf("existing file was changed: %q", data)
	}
}

func TestWriteFileAtomicReplacesWithoutLeavingTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := WriteFileAtomic(path, []byte("v1"), 0o600); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteFileAtomic(path, []byte("v2"), 0o600); err != nil {
		t.Fatalf("replacement: %v", err)
	}
	data, _ := os.ReadFile(path) //nolint:gosec // G304: path inside t.TempDir() of this test
	if string(data) != "v2" {
		t.Fatalf("content = %q, want v2", data)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("directory has %d entries, want only the final file", len(entries))
	}
}

func TestWriteFileAtomicAppliesRestrictivePermission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := WriteFileAtomic(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Windows does not expose POSIX permission bits through os.FileMode; the assertion applies to the other systems.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("permission = %o, want 600", info.Mode().Perm())
		}
	}
}
