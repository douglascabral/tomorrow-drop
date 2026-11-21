package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tomorrow-drop/internal/transfer"
)

// A picture from the clipboard is sent as a file, named by when it was taken.
func TestClipboardImageBecomesAFileToSend(t *testing.T) {
	g := newTestGUI(t)
	png := []byte("\x89PNG\r\n\x1a\nfake-image-bytes")
	spec, err := g.imageSpec(png, ".png")
	if err != nil {
		t.Fatalf("imageSpec: %v", err)
	}
	if spec.Kind != transfer.KindFile || len(spec.Paths) != 1 {
		t.Fatalf("spec = %+v, want one file", spec)
	}
	name := filepath.Base(spec.Paths[0])
	if !strings.HasPrefix(name, "Captura-") || !strings.HasSuffix(name, ".png") {
		t.Fatalf("file name = %q, want Captura-<date>.png", name)
	}
	data, err := os.ReadFile(spec.Paths[0]) //nolint:gosec // G304: file in a temporary folder of the test
	if err != nil || !bytes.Equal(data, png) {
		t.Fatalf("staged content differs from the clipboard image (%v)", err)
	}
}

// Only picture formats are sent; anything else from the clipboard is refused.
func TestClipboardImageFormatsAreChecked(t *testing.T) {
	cases := map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/gif":  ".gif",
		"image/webp": ".webp",
		"text/html":  "",
		"":           "",
	}
	for mime, want := range cases {
		if got := clipboardExt(mime); got != want {
			t.Errorf("clipboardExt(%q) = %q, want %q", mime, got, want)
		}
	}
}

// An empty picture or one of an unknown format is not sent.
func TestClipboardImageRefusesEmptyAndUnknown(t *testing.T) {
	g := newTestGUI(t)
	if _, err := g.imageSpec(nil, ".png"); err == nil {
		t.Errorf("an empty picture must be refused")
	}
	if _, err := g.imageSpec([]byte("x"), ".exe"); err == nil {
		t.Errorf("an unknown format must be refused")
	}
}
