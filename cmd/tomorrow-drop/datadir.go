package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tomorrow-drop/internal/storage"
)

// dataDirFrom picks the folder of the application's files (identity, trusted
// devices, settings). The user configuration folder is used when the system has
// one. Phones may have none, so the folder that the application storage gives
// is used then. The temporary folder is only the last resort, because the
// system may delete it.
func dataDirFrom(userConfig func() (string, error), appStorage func() string) string {
	if dir, err := userConfig(); err == nil && dir != "" {
		return filepath.Join(dir, "tomorrow-drop")
	}
	if dir := appStorage(); dir != "" {
		return filepath.Join(dir, "tomorrow-drop")
	}
	return filepath.Join(os.TempDir(), "tomorrow-drop")
}

// legacyFiles are the files of the earlier location that must keep their
// content: the identity and the trusted devices.
var legacyFiles = []string{"identity.key", "trust.json"}

// migrateHome copies the identity and the trusted devices from an earlier folder
// into the data folder. A file is copied only when the data folder does not
// have the identity yet, so a later run never replaces the identity in use.
func migrateHome(from, to string) error {
	if from == to {
		return nil
	}
	if _, err := os.Stat(filepath.Join(to, "identity.key")); err == nil {
		return nil
	}
	for _, name := range legacyFiles {
		data, err := os.ReadFile(filepath.Join(from, name)) //nolint:gosec // G304: files of the application's own folder
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := storage.WriteFileAtomic(filepath.Join(to, name), data, 0o600); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
	}
	return nil
}
