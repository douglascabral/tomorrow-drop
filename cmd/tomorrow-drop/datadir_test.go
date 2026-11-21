package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression: on Android os.UserConfigDir fails, so the settings path was empty
// and choosing a language failed. The app's own storage must be used instead.
func TestDataDirUsesStorageWhenUserConfigIsMissing(t *testing.T) {
	noConfig := func() (string, error) { return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined") }
	storage := func() string { return "/data/user/0/org.tomorrowdrop.app/files" }
	got := dataDirFrom(noConfig, storage)
	want := filepath.Join("/data/user/0/org.tomorrowdrop.app/files", "tomorrow-drop")
	if got != want {
		t.Fatalf("dataDirFrom = %q, want %q", got, want)
	}
}

func TestDataDirKeepsUserConfigWhenAvailable(t *testing.T) {
	config := func() (string, error) { return "/home/u/.config", nil }
	storage := func() string { t.Fatal("storage must not be asked when the config folder exists"); return "" }
	if got, want := dataDirFrom(config, storage), filepath.Join("/home/u/.config", "tomorrow-drop"); got != want {
		t.Fatalf("dataDirFrom = %q, want %q", got, want)
	}
}

func TestDataDirNeverEmpty(t *testing.T) {
	fail := func() (string, error) { return "", errors.New("no config") }
	none := func() string { return "" }
	if got := dataDirFrom(fail, none); !strings.HasSuffix(got, "tomorrow-drop") {
		t.Fatalf("dataDirFrom = %q, want a tomorrow-drop folder", got)
	}
}

// Files written by earlier versions into the temporary folder (the Android
// cache) move to the data folder, so the identity and trusted devices survive.
func TestMigrateCopiesIdentityAndTrustOnce(t *testing.T) {
	old := t.TempDir()
	dst := t.TempDir()
	if err := os.WriteFile(filepath.Join(old, "identity.key"), []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "trust.json"), []byte("trust"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := migrateHome(old, dst); err != nil {
		t.Fatalf("migrateHome: %v", err)
	}
	for name, want := range map[string]string{"identity.key": "key", "trust.json": "trust"} {
		data, err := os.ReadFile(filepath.Join(dst, name)) //nolint:gosec // G304: test file in a temporary folder
		if err != nil || string(data) != want {
			t.Fatalf("%s after migration = %q (%v), want %q", name, data, err, want)
		}
	}

	// A later run must not overwrite the identity already in use.
	if err := os.WriteFile(filepath.Join(old, "identity.key"), []byte("older"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := migrateHome(old, dst); err != nil {
		t.Fatalf("second migrateHome: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dst, "identity.key")) //nolint:gosec // G304: test file in a temporary folder
	if string(data) != "key" {
		t.Fatalf("identity was overwritten: %q", data)
	}
}
