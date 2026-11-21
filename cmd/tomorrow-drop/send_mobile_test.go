package main

import (
	"testing"
)

// On a phone the drop hint must not talk about a computer, and the send action
// is a large labeled button, as on the other screens.
func TestSendScreenOnPhoneSpeaksOfThisDevice(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = true
	g.showSend("", false)
	labels := labelTexts(g.win.Content())
	if contains(labels, g.tr.T("gui.send.drop_hint")) {
		t.Fatalf("phone shows the computer hint: %q", g.tr.T("gui.send.drop_hint"))
	}
	if !contains(labels, g.tr.T("gui.send.pick_hint_mobile")) {
		t.Fatalf("phone lacks its own hint; texts = %v", labels)
	}
}

func TestSendButtonOnPhoneIsLabeledAndLarge(t *testing.T) {
	g := newTestGUI(t)
	g.app.Settings().SetTheme(themeFor(true))
	g.mobile = true
	g.showSend("", false)
	button := findButton(g.win.Content(), g.tr.T("gui.send.send"))
	if button == nil {
		t.Fatalf("phone send screen has no labeled send button; buttons = %v", buttonLabels(g.win.Content()))
	}
	if h := button.MinSize().Height; h < touchMinHeight {
		t.Fatalf("send button height = %v, want at least %v", h, touchMinHeight)
	}
	if w := button.Size().Width; w > 0 && w < 200 {
		t.Fatalf("send button is %v wide on a phone; it should span the screen", w)
	}
}
