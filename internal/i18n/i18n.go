// Package i18n translates the texts shown to the user.
//
// The language is chosen as follows: the person's explicit preference; then
// the operating system language; then English (Fallback). Keys missing from
// the chosen language use the English text.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Fallback is the language used when no other supported language applies.
const Fallback = "en"

// Supported lists the available languages. pt-BR is the initial language.
var Supported = []string{"pt-BR", Fallback}

//go:embed locales/*.json
var translationFiles embed.FS

// Catalog maps language → key → text.
type Catalog map[string]map[string]string

// Bundle translates texts in one language, falling back to Fallback.
type Bundle struct {
	lang    string
	catalog Catalog
}

// NewBundle creates a translator for the given language and catalogs.
func NewBundle(lang string, catalog Catalog) *Bundle {
	return &Bundle{lang: lang, catalog: catalog}
}

// Load creates a translator from the catalogs embedded in the binary.
func Load(lang string) (*Bundle, error) {
	catalog := Catalog{}
	for _, code := range Supported {
		data, err := fs.ReadFile(translationFiles, "locales/"+code+".json")
		if err != nil {
			return nil, fmt.Errorf("read translations for %s: %w", code, err)
		}
		var texts map[string]string
		if err := json.Unmarshal(data, &texts); err != nil {
			return nil, fmt.Errorf("translations for %s: %w", code, err)
		}
		catalog[code] = texts
	}
	return NewBundle(lang, catalog), nil
}

// T returns the text for the key. With args, it applies fmt.Sprintf. If the
// key exists in no language, it returns the key itself.
func (b *Bundle) T(key string, args ...any) string {
	text, ok := b.catalog[b.lang][key]
	if !ok {
		text, ok = b.catalog[Fallback][key]
	}
	if !ok {
		return key
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

// Match converts a locale tag (pt_BR.UTF-8, en-US, fr) into a supported
// language. pt-PT and other Portuguese variants map to pt-BR. It returns ""
// when no supported language matches.
func Match(tag string) string {
	base := tag
	if i := strings.IndexAny(base, ".@"); i >= 0 {
		base = base[:i]
	}
	base = strings.ToLower(strings.ReplaceAll(base, "_", "-"))
	language, _, _ := strings.Cut(base, "-")
	switch language {
	case "pt":
		return "pt-BR"
	case "en":
		return Fallback
	default:
		return ""
	}
}

// DetectFromEnv looks for a supported language in the locale environment
// variables. LANGUAGE may hold a colon-separated list. The order is LANGUAGE,
// LC_ALL, LC_MESSAGES and LANG; the first supported value wins.
func DetectFromEnv(get func(string) string) string {
	candidates := strings.Split(get("LANGUAGE"), ":")
	candidates = append(candidates, get("LC_ALL"), get("LC_MESSAGES"), get("LANG"))
	for _, c := range candidates {
		if lang := Match(c); lang != "" {
			return lang
		}
	}
	return ""
}

// DetectSystem returns the operating system language, or "" if there is none.
func DetectSystem() string {
	if lang := Match(SystemLocale()); lang != "" {
		return lang
	}
	return DetectFromEnv(os.Getenv)
}

// ValidPreference reports whether the preference is "auto" or a supported language.
func ValidPreference(p string) bool {
	if p == "auto" {
		return true
	}
	return isSupported(p)
}

// Resolve picks the final language. A supported preference wins; "auto" or an
// invalid value uses the detected language; with nothing supported, it uses Fallback.
func Resolve(preference, detected string) string {
	if preference != "" && preference != "auto" && isSupported(preference) {
		return preference
	}
	if lang := Match(detected); lang != "" {
		return lang
	}
	return Fallback
}

func isSupported(lang string) bool {
	for _, s := range Supported {
		if s == lang {
			return true
		}
	}
	return false
}
