package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// Regression: on a phone the chosen file is a content address, not a path, so
// the file was read as empty and failed. The content is copied into the app's
// folder when it is chosen, under its own name.
func TestPickedFileIsCopiedWithItsContent(t *testing.T) {
	dir := t.TempDir()
	path, err := stagePicked(strings.NewReader("conteúdo do arquivo"), "relatório.pdf", dir)
	if err != nil {
		t.Fatalf("stagePicked: %v", err)
	}
	if filepath.Base(path) != "relatório.pdf" {
		t.Fatalf("staged name = %q, want the original name", filepath.Base(path))
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: test file in a temporary folder
	if err != nil {
		t.Fatalf("read staged file: %v", err)
	}
	if string(data) != "conteúdo do arquivo" {
		t.Fatalf("staged content = %q, want the chosen file's content", data)
	}
}

func TestPickedFileWithUnsafeNameIsRefused(t *testing.T) {
	if _, err := stagePicked(strings.NewReader("x"), "../escape", t.TempDir()); err == nil {
		t.Fatalf("a name that leaves the folder must be refused")
	}
}

// The file dialog uses most of the window, so it grows with a maximized window.
func TestPickerFillsTheWindow(t *testing.T) {
	big := pickerSize(fyne.NewSize(1920, 1080))
	if big.Width < 1500 || big.Height < 900 {
		t.Fatalf("picker on a large window = %v, want most of the window", big)
	}
	small := pickerSize(fyne.NewSize(420, 640))
	if small.Width > 420 || small.Height > 640 {
		t.Fatalf("picker on a small window = %v, larger than the window", small)
	}
}
