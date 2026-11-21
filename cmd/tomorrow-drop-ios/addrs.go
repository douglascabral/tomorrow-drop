package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// The last address of each paired device, by fingerprint, kept in the
// application's folder. It names a device that discovery finds at that address;
// trust still needs an authenticated connection.
const addrsFile = "paired-addresses.json"

func loadAddrs(home string) map[string]string {
	m := map[string]string{}
	data, err := os.ReadFile(filepath.Join(home, addrsFile)) //nolint:gosec // G304: file in the application's folder
	if err != nil {
		return m
	}
	if json.Unmarshal(data, &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

func saveAddrs(home string, m map[string]string) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(home, addrsFile), data, 0o600)
}

func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
