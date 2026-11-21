//go:build darwin

package i18n

import (
	"os/exec"
	"strings"
)

// SystemLocale reads the macOS user locale. Apps opened from Finder usually
// do not inherit LANG, so the system preference is queried instead.
func SystemLocale() string {
	output, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
