package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxDeviceLabelRunes matches the limit of a discovery name (trust.ValidateName).
const maxDeviceLabelRunes = 64

// DeviceLabel joins the device name and its operating system for the
// announcement, for example "iPad de Douglas · iPadOS 26.1". Empty parts are
// left out. Control characters become spaces, runs of spaces collapse, and the
// result is cut to the discovery name limit. An empty result means the caller
// must use another name.
func DeviceLabel(name, system string) string {
	var parts []string
	for _, p := range []string{name, system} {
		if clean := cleanLabel(p); clean != "" {
			parts = append(parts, clean)
		}
	}
	label := strings.Join(parts, " · ")
	if utf8.RuneCountInString(label) > maxDeviceLabelRunes {
		runes := []rune(label)
		label = strings.TrimSpace(string(runes[:maxDeviceLabelRunes]))
	}
	return label
}

func cleanLabel(s string) string {
	mapped := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(mapped), " ")
}
