package main

import (
	"strings"
	"testing"

	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/ui"
)

func withLanguage(t *testing.T, g *gui, lang string) {
	t.Helper()
	tr, err := i18n.Load(lang)
	if err != nil {
		t.Fatal(err)
	}
	g.tr = tr
}

// "1 file(s)" reads badly: counts agree with the number in both languages.
func TestCountsAgreeWithTheNumber(t *testing.T) {
	g := newTestGUI(t)
	cases := map[string]map[int]string{
		"en":    {0: "0 files", 1: "1 file", 2: "2 files"},
		"pt-BR": {0: "0 arquivos", 1: "1 arquivo", 2: "2 arquivos"},
	}
	for lang, byN := range cases {
		withLanguage(t, g, lang)
		for n, want := range byN {
			if got := g.countFiles(n); got != want {
				t.Errorf("%s countFiles(%d) = %q, want %q", lang, n, got, want)
			}
		}
	}
	withLanguage(t, g, "en")
	if g.countItems(1) != "1 item" || g.countItems(3) != "3 items" {
		t.Errorf("countItems = %q, %q", g.countItems(1), g.countItems(3))
	}
}

// Files received on a phone are saved where the person chose, not in Downloads.
func TestSavedMessageMatchesThePlatform(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = false
	if got := g.savedText(2); !strings.Contains(got, "Downloads") || !strings.Contains(got, "2 files") {
		t.Errorf("desktop saved text = %q", got)
	}
	g.mobile = true
	if got := g.savedText(2); strings.Contains(got, "Downloads") || !strings.Contains(got, "2 files") {
		t.Errorf("phone saved text = %q, must not mention Downloads", got)
	}
}

// The request screen asks a question; it does not say the files were received.
func TestOfferTextsAreAboutARequest(t *testing.T) {
	g := newTestGUI(t)
	if got := g.tr.T("gui.receive.title"); strings.Contains(got, "%") || got != "Receive files?" {
		t.Errorf("offer title = %q", got)
	}
	if got := g.tr.T("gui.receive.from_device", "laptop"); got != "laptop wants to send you:" {
		t.Errorf("offer heading = %q", got)
	}
}

// The port is a detail nobody acts on; the waiting screen does not show it.
func TestReceiveScreenDoesNotShowThePort(t *testing.T) {
	g := newTestGUI(t)
	g.showReceive()
	for _, text := range labelTexts(g.win.Content()) {
		if strings.Contains(text, "47521") {
			t.Fatalf("receive screen shows the port: %q", text)
		}
	}
}

// A failed transfer does not sit under a header that says it is complete.
func TestResultHeaderIsNeutral(t *testing.T) {
	g := newTestGUI(t)
	g.showOutcome(false, g.tr.T("gui.send.failed"), "x", nil)
	for _, text := range labelTexts(g.win.Content()) {
		if strings.Contains(strings.ToLower(text), "complete") {
			t.Fatalf("failed result screen says %q", text)
		}
	}
}

// After a contact the application knows the device by the name the person gave.
func TestDisplayNamePrefersThePairedName(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	if got := g.displayName(ep.String()); got != ep.String() {
		t.Fatalf("before contact displayName = %q, want the address", got)
	}
	g.openDevice(ui.DeviceRow{Name: "x", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))
	if got := g.displayName(ep.String()); got != "other" {
		t.Fatalf("displayName = %q, want the paired name", got)
	}
}
