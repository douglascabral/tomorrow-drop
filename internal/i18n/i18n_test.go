package i18n

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func environment(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestMatchNormalizesLocaleTags(t *testing.T) {
	cases := map[string]string{
		"pt_BR.UTF-8": "pt-BR",
		"pt-br":       "pt-BR",
		"pt":          "pt-BR",
		"pt_PT":       "pt-BR",
		"en_US.UTF-8": "en",
		"en":          "en",
		"fr_FR":       "",
		"C":           "",
		"POSIX":       "",
		"":            "",
	}
	for input, want := range cases {
		if got := Match(input); got != want {
			t.Errorf("Match(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDetectFromEnvFollowsVariablePriority(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"LC_ALL takes priority over LANG", map[string]string{"LC_ALL": "pt_BR.UTF-8", "LANG": "en_US.UTF-8"}, "pt-BR"},
		{"LANG when LC_ALL is missing", map[string]string{"LANG": "en_US.UTF-8"}, "en"},
		{"LANGUAGE accepts an ordered list", map[string]string{"LANG": "fr_FR", "LANGUAGE": "fr:pt_BR:en"}, "pt-BR"},
		{"unsupported locale returns empty", map[string]string{"LANG": "fr_FR.UTF-8"}, ""},
		{"no variables returns empty", map[string]string{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectFromEnv(environment(c.values)); got != c.want {
				t.Fatalf("DetectFromEnv = %q, want %q", got, c.want)
			}
		})
	}
}

func TestResolveExplicitPreferenceOverridesDetection(t *testing.T) {
	if got := Resolve("en", "pt-BR"); got != "en" {
		t.Fatalf("explicit preference did not win: %q", got)
	}
}

func TestResolveAutoUsesDetection(t *testing.T) {
	for _, pref := range []string{"", "auto"} {
		if got := Resolve(pref, "pt-BR"); got != "pt-BR" {
			t.Fatalf("Resolve(%q, pt-BR) = %q, want pt-BR", pref, got)
		}
	}
}

func TestResolveFallsBackToEnglishForUnsupportedLanguage(t *testing.T) {
	if got := Resolve("auto", "fr"); got != Fallback {
		t.Fatalf("Resolve with unsupported language = %q, want %q", got, Fallback)
	}
	if got := Resolve("", ""); got != Fallback {
		t.Fatalf("Resolve without information = %q, want %q", got, Fallback)
	}
}

func TestResolveIgnoresInvalidPreference(t *testing.T) {
	if got := Resolve("klingon", "pt-BR"); got != "pt-BR" {
		t.Fatalf("invalid preference was not ignored: %q", got)
	}
}

func TestTranslatorUsesChosenLanguage(t *testing.T) {
	catalog := Catalog{
		"pt-BR": {"greeting": "olá"},
		"en":    {"greeting": "hello"},
	}
	tr := NewBundle("pt-BR", catalog)
	if got := tr.T("greeting"); got != "olá" {
		t.Fatalf("T = %q, want olá", got)
	}
}

func TestTranslatorFallsBackToEnglishWhenKeyMissingInLanguage(t *testing.T) {
	catalog := Catalog{
		"pt-BR": {},
		"en":    {"english_only": "english only"},
	}
	tr := NewBundle("pt-BR", catalog)
	if got := tr.T("english_only"); got != "english only" {
		t.Fatalf("fallback to en = %q, want %q", got, "english only")
	}
}

func TestTranslatorReturnsKeyWhenMissingEverywhere(t *testing.T) {
	tr := NewBundle("pt-BR", Catalog{"pt-BR": {}, "en": {}})
	if got := tr.T("missing"); got != "missing" {
		t.Fatalf("missing key = %q, want the key itself", got)
	}
}

func TestTranslatorAppliesArguments(t *testing.T) {
	tr := NewBundle("en", Catalog{"en": {"hi": "hello %s"}})
	if got := tr.T("hi", "Ana"); got != "hello Ana" {
		t.Fatalf("T with argument = %q, want %q", got, "hello Ana")
	}
}

func TestEmbeddedCatalogsHaveSameKeys(t *testing.T) {
	pt := loadEmbedded(t, "pt-BR")
	en := loadEmbedded(t, "en")

	for key := range en {
		if _, ok := pt[key]; !ok {
			t.Errorf("key %q exists in en but is missing from pt-BR", key)
		}
	}
	for key := range pt {
		if _, ok := en[key]; !ok {
			t.Errorf("key %q exists in pt-BR but is missing from en", key)
		}
	}
	for key, value := range en {
		if value == "" {
			t.Errorf("empty value for %q in en", key)
		}
	}
}

func TestDefaultBundleLoadsPortugueseTranslation(t *testing.T) {
	tr, err := Load("pt-BR")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := tr.T("app.nome"); got != "Tomorrow Drop" {
		t.Fatalf("app.nome = %q", got)
	}
	if got := tr.T("cli.versao"); got == "cli.versao" || got == "" {
		t.Fatalf("cli.versao not translated in pt-BR: %q", got)
	}
}

func TestDetectSystemReturnsSupportedLanguageOrEmpty(t *testing.T) {
	// The real reading depends on the system, so the test checks the contract:
	// a supported language or empty (which leads to the fallback in Resolve).
	lang := DetectSystem()
	if lang != "" && !isSupported(lang) {
		t.Fatalf("DetectSystem = %q, is not a supported language", lang)
	}
}

func loadEmbedded(t *testing.T, lang string) map[string]string {
	t.Helper()
	data, err := fs.ReadFile(translationFiles, "locales/"+lang+".json")
	if err != nil {
		t.Fatalf("read %s: %v", lang, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("JSON of %s: %v", lang, err)
	}
	return m
}

// A placeholder that differs between languages shows "%!s(MISSING)" or "%!(EXTRA
// ...)" at run time, so every key must use the same placeholders in order.
func TestEmbeddedCatalogsUseTheSamePlaceholders(t *testing.T) {
	pt := loadEmbedded(t, "pt-BR")
	en := loadEmbedded(t, "en")
	verbs := regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)
	for key, enValue := range en {
		ptValue, ok := pt[key]
		if !ok {
			continue
		}
		if got, want := verbs.FindAllString(ptValue, -1), verbs.FindAllString(enValue, -1); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("key %q: pt-BR uses %v, en uses %v", key, got, want)
		}
	}
}

// A counted noun agrees with the number: the singular for one, the plural with
// the number otherwise.
func TestCountAgreesWithTheNumber(t *testing.T) {
	cases := map[string]map[int]string{
		"en":    {0: "0 files", 1: "1 file", 2: "2 files"},
		"pt-BR": {0: "0 arquivos", 1: "1 arquivo", 2: "2 arquivos"},
	}
	for lang, byN := range cases {
		tr, err := Load(lang)
		if err != nil {
			t.Fatal(err)
		}
		for n, want := range byN {
			if got := tr.Count("count.file", n); got != want {
				t.Errorf("%s Count(%d) = %q, want %q", lang, n, got, want)
			}
		}
	}
}
