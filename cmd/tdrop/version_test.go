package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The version is written in several places: the VERSION file is the source,
// and the files that cannot read it (the Fyne metadata and the iOS plist) must
// say the same. A release tag must match all of them.
func TestVersionIsTheSameEverywhere(t *testing.T) {
	want := readVersionFile(t, "../../VERSION")
	checks := map[string]*regexp.Regexp{
		"../../FyneApp.toml":                regexp.MustCompile(`(?m)^\s*Version = "([^"]+)"`),
		"../../ios/TomorrowDrop/Info.plist": regexp.MustCompile(`CFBundleShortVersionString</key>\s*<string>([^<]+)</string>`),
	}
	for file, re := range checks {
		data, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatal(err)
		}
		m := re.FindSubmatch(data)
		if m == nil {
			t.Fatalf("%s has no version", file)
		}
		if got := string(m[1]); got != want {
			t.Errorf("%s says %q, VERSION says %q", file, got, want)
		}
	}
}

func readVersionFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	v := strings.TrimSpace(string(data))
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(v) {
		t.Fatalf("VERSION must be MAJOR.MINOR.PATCH, got %q", v)
	}
	return v
}
