// Package assets embeds the visual resources of the application (PRD §19.2).
package assets

import _ "embed"

//go:embed logo.svg
var logoSVG []byte

// LogoSVG returns the logo: two mountains and a sun.
func LogoSVG() []byte {
	return logoSVG
}
