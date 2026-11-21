package main

import (
	"testing"

	"fyne.io/fyne/v2"
)

// Regression: on a phone the manual pairing screen was wider than the window,
// because its help text did not wrap. Every screen must fit the phone width.
func TestScreensFitThePhoneWidth(t *testing.T) {
	const phoneWidth = 420
	screens := map[string]func(g *gui){
		"home":     func(g *gui) { g.showHome() },
		"manual":   func(g *gui) { g.showManual() },
		"settings": func(g *gui) { g.showSettings() },
		"trusted":  func(g *gui) { g.showTrusted() },
		"send":     func(g *gui) { g.showSend("", false) },
		"receive":  func(g *gui) { g.showReceive() },
		"text":     func(g *gui) { g.showText("uma frase longa que precisa quebrar linha para caber na tela do celular") },
		"completed": func(g *gui) {
			g.showCompleted("Recebido", "2 arquivos salvos em Downloads", []string{"arquivo-com-nome-muito-longo-para-a-tela.pdf"}, g.receiveAgainAction())
		},
	}
	for name, show := range screens {
		g := newTestGUI(t)
		g.app.Settings().SetTheme(themeFor(true))
		g.mobile = true
		show(g)
		g.win.Resize(fyne.NewSize(phoneWidth, 640))
		if w := g.win.Content().MinSize().Width; w > phoneWidth {
			t.Errorf("%s screen needs %v pixels, the phone window has %v", name, w, phoneWidth)
		}
	}
}

// Labels wrap, so long text never pushes a screen wider than its window.
func TestLabelsWrapText(t *testing.T) {
	if l := newLabel("texto"); l.Wrapping != fyne.TextWrapWord {
		t.Fatalf("newLabel wrapping = %v, want TextWrapWord", l.Wrapping)
	}
	if l := newLabelStyled("texto", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}); l.Wrapping != fyne.TextWrapWord || !l.TextStyle.Bold {
		t.Fatalf("newLabelStyled must wrap and keep its style")
	}
}

// Regression: the receive screen title showed the unformatted "Incoming %s".
func TestReceiveScreenHasAPlainTitle(t *testing.T) {
	g := newTestGUI(t)
	g.showReceive()
	for _, text := range labelTexts(g.win.Content()) {
		if containsPlaceholder(text) {
			t.Fatalf("receive screen shows an unformatted placeholder: %q", text)
		}
	}
}

func containsPlaceholder(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '%' {
			return true
		}
	}
	return false
}

// Regression: the status of a device row is one short word on the right; it
// must not wrap, or it shows a single letter. The name wraps.
func TestDeviceRowStatusStaysOnOneLine(t *testing.T) {
	g := newTestGUI(t)
	row := g.newDeviceRow()
	parts := g.rowParts[row]
	if parts == nil {
		t.Fatalf("device row has no parts")
	}
	if parts.status.Wrapping != fyne.TextWrapOff {
		t.Fatalf("status wrapping = %v, want TextWrapOff", parts.status.Wrapping)
	}
	if parts.name.Wrapping != fyne.TextWrapWord {
		t.Fatalf("name wrapping = %v, want TextWrapWord", parts.name.Wrapping)
	}
}
