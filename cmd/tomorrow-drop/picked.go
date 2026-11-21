package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"

	"tomorrow-drop/internal/transfer"
)

// stagePicked copies a file chosen on a phone into its own folder under dir and
// returns the path of the copy. The phone gives the content through a reader,
// not a path. Each file gets a new folder, so a file never replaces another with
// the same name, and the name is checked like any file name of an offer.
func stagePicked(r io.Reader, name, dir string) (string, error) {
	if err := transfer.ValidateName(name); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create folder for chosen files: %w", err)
	}
	folder, err := os.MkdirTemp(dir, "pick-")
	if err != nil {
		return "", fmt.Errorf("create folder for the chosen file: %w", err)
	}
	path := filepath.Join(folder, name)
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: folder created above
	if err != nil {
		return "", fmt.Errorf("create chosen file: %w", err)
	}
	if _, err := io.Copy(out, r); err != nil {
		_ = out.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("copy chosen file: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("finish chosen file: %w", err)
	}
	return path, nil
}

// pickerSize is the size of the file dialog: most of the window, so the dialog
// grows with a maximized window and never exceeds a small one.
func pickerSize(win fyne.Size) fyne.Size {
	const share = 0.9
	return fyne.NewSize(win.Width*share, win.Height*share)
}

// droppedPaths keeps the regular files among the items dropped on the window.
// Folders, missing files and addresses that are not files are ignored.
func droppedPaths(uris []fyne.URI) []string {
	var paths []string
	for _, u := range uris {
		if u.Scheme() != "file" {
			continue
		}
		info, err := os.Stat(u.Path())
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		paths = append(paths, u.Path())
	}
	return paths
}
