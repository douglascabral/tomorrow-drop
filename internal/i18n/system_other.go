//go:build !windows && !darwin

package i18n

// SystemLocale returns empty: on Linux and other Unix systems the language
// comes from the environment variables, handled by DetectFromEnv.
func SystemLocale() string {
	return ""
}
