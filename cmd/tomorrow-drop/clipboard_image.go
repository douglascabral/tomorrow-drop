package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/transfer"
)

// clipboardExt is the file extension of a picture of the given MIME type, or ""
// when the type is not a picture format that is sent.
func clipboardExt(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	}
	return ""
}

// imageSpec stages a picture from the clipboard as a file and returns the send
// that carries it. The picture is named by when it was taken, for example
// "Captura-2026-11-21-10-45-03.png", so it reads well on the other device.
func (g *gui) imageSpec(data []byte, ext string) (transfer.SendSpec, error) {
	if len(data) == 0 {
		return transfer.SendSpec{}, fmt.Errorf("clipboard picture is empty: %w", apperr.ErrMalformed)
	}
	if int64(len(data)) > limits.MaxFileBytes {
		return transfer.SendSpec{}, fmt.Errorf("clipboard picture is larger than the limit: %w", apperr.ErrLimitExceeded)
	}
	if !isPictureExt(ext) {
		return transfer.SendSpec{}, fmt.Errorf("clipboard picture format %q is not sent: %w", ext, apperr.ErrMalformed)
	}
	name := "Captura-" + time.Now().Format("2006-01-02-15-04-05") + ext
	path, err := stagePicked(bytes.NewReader(data), name, filepath.Join(g.home, "clipboard"))
	if err != nil {
		return transfer.SendSpec{}, err
	}
	return transfer.SendSpec{Kind: transfer.KindFile, Paths: []string{path}}, nil
}

// isPictureExt reports whether ext is an extension of a picture format that is sent.
func isPictureExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".gif", ".webp":
		return true
	}
	return false
}
