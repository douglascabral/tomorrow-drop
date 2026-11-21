package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Regression: entering the settings showed "Saved. The new language applies..."
// at once, because setting the current value fired the change handler. Showing
// the current language must not save or notify anything.
func TestLanguagePickerShowsCurrentLanguageWithoutSaving(t *testing.T) {
	saved := false
	picker := languagePicker("en", "Automatic", func(string) { saved = true })
	if picker.Selected != "English" {
		t.Fatalf("picker shows %q, want the current language English", picker.Selected)
	}
	if saved {
		t.Fatalf("showing the current language saved or notified")
	}
}

// Choosing another language still saves it.
func TestLanguagePickerSavesAChoice(t *testing.T) {
	var got string
	picker := languagePicker("auto", "Automatic", func(c string) { got = c })
	picker.SetSelected("Português (Brasil)")
	if got != "pt-BR" {
		t.Fatalf("choice = %q, want pt-BR", got)
	}
}

// The "about" text is long: it must wrap on a narrow window instead of being cut.
func TestSettingsAboutTextWraps(t *testing.T) {
	g := newTestGUI(t)
	g.showSettings()
	about := g.tr.T("gui.settings.about")
	label := findLabel(g.win.Content(), about)
	if label == nil {
		t.Fatalf("about text not found on the settings screen")
	}
	if label.Wrapping != fyne.TextWrapWord {
		t.Fatalf("about text wrapping = %v, want TextWrapWord", label.Wrapping)
	}
}

// findLabel returns the label with the text under obj, or nil.
func findLabel(obj fyne.CanvasObject, text string) *widget.Label {
	switch o := obj.(type) {
	case *widget.Label:
		if o.Text == text {
			return o
		}
	case *container.Scroll:
		return findLabel(o.Content, text)
	case *container.ThemeOverride:
		return findLabel(o.Content, text)
	case *fyne.Container:
		for _, c := range o.Objects {
			if l := findLabel(c, text); l != nil {
				return l
			}
		}
	}
	return nil
}
