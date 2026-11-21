package transfer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tomorrow-drop/internal/apperr"
)

const (
	stagingPattern = "tdrop-*.part"
	stagingMatch   = "tdrop-*.part"
)

// publishExclusive moves a staged file into dir under name, or under the
// first free "name (n).ext" variant. It never overwrites an existing file. The
// link is atomic: the file either appears completely or not at all.
func publishExclusive(staged, dir, name string) (string, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("destination directory %q is not available: %w", dir, apperr.ErrMalformed)
	}
	base, ext := splitExt(name)
	for i := 0; i < 10000; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		final := filepath.Join(dir, candidate)
		err := os.Link(staged, final)
		if err == nil {
			_ = os.Remove(staged)
			return final, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return "", fmt.Errorf("publish %s: %w", candidate, err)
	}
	return "", fmt.Errorf("no free name for %s: %w", name, apperr.ErrLimitExceeded)
}

func splitExt(name string) (string, string) {
	ext := filepath.Ext(name)
	if ext == name {
		return name, ""
	}
	return name[:len(name)-len(ext)], ext
}

// CleanupAbandoned removes staging files left by an unexpected shutdown. Only
// files matching the application's own pattern and older than olderThan are
// removed; any other file in the directory is kept.
func CleanupAbandoned(dir string, olderThan time.Duration, now time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read staging directory: %w", err)
	}
	removed := 0
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		if ok, _ := filepath.Match(stagingMatch, e.Name()); !ok {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) < olderThan {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
			removed++
		}
	}
	return removed, nil
}
