package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The announced name is the device name and its system, joined for people to
// read ("iPad de Douglas · iPadOS 26.1"). It must pass the rules of the
// discovery name: not empty, at most 64 characters, no control characters.
func TestDeviceLabelJoinsNonEmptyParts(t *testing.T) {
	cases := []struct {
		name, system, want string
	}{
		{"iPad de Douglas", "iPadOS 26.1", "iPad de Douglas · iPadOS 26.1"},
		{"", "Android 15", "Android 15"},
		{"samsung SM-S911B", "", "samsung SM-S911B"},
		{"  ", "  ", ""},
	}
	for _, c := range cases {
		if got := DeviceLabel(c.name, c.system); got != c.want {
			t.Errorf("DeviceLabel(%q, %q) = %q, want %q", c.name, c.system, got, c.want)
		}
	}
}

func TestDeviceLabelReplacesControlCharactersAndCollapsesSpaces(t *testing.T) {
	got := DeviceLabel("pad\tof\x00 Douglas", "")
	if got != "pad of Douglas" {
		t.Fatalf("DeviceLabel = %q, want control characters replaced by single spaces", got)
	}
}

func TestDeviceLabelFitsTheDiscoveryNameLimit(t *testing.T) {
	long := strings.Repeat("é", 100)
	got := DeviceLabel(long, "iPadOS 26.1")
	if n := utf8.RuneCountInString(got); n > 64 {
		t.Fatalf("label has %d characters, want at most 64", n)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("truncation split a character: %q", got)
	}
}
