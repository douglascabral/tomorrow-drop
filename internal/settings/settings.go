// Package settings stores the user's preferences in a JSON file.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/storage"
)

// Settings are the persisted preferences.
type Settings struct {
	// Language is "auto" (the system language) or a supported language.
	Language string `json:"language"`
}

// Default holds the preferences used when there is no saved file.
var Default = Settings{Language: "auto"}

// DefaultPath returns the path of the user's settings file.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate configuration directory: %w", err)
	}
	return filepath.Join(dir, "tomorrow-drop", "settings.json"), nil
}

// Load reads the preferences. A missing file yields Default; an invalid file
// yields apperr.ErrMalformed.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: settings file in the user's own configuration directory
	if os.IsNotExist(err) {
		return Default, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}
	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("invalid settings: %w: %w", apperr.ErrMalformed, err)
	}
	if !i18n.ValidPreference(cfg.Language) {
		return Settings{}, fmt.Errorf("invalid language in settings: %w", apperr.ErrMalformed)
	}
	return cfg, nil
}

// Save writes the preferences atomically.
func Save(path string, cfg Settings) error {
	if !i18n.ValidPreference(cfg.Language) {
		return fmt.Errorf("language %q: %w", cfg.Language, apperr.ErrMalformed)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	return storage.WriteFileAtomic(path, data, 0o600)
}
