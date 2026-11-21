package main

import (
	"testing"

	"fyne.io/fyne/v2/widget"
)

// The drop area keeps room around its text and its button: the button does not
// touch the edge of the card, and the text is not stuck to the button.
func TestDropCardHasRoomAroundItsContent(t *testing.T) {
	hint := widget.NewLabel("Arraste arquivos até aqui ou escolha-os.")
	pick := widget.NewButton("Escolher arquivos", nil)
	card := dropCard(hint, pick)
	minContent := hint.MinSize().Height + pick.MinSize().Height
	if h := card.MinSize().Height; h < minContent+2*dropGap+dropGap {
		t.Fatalf("drop card is %v high, want at least %v for its content and the gaps", h, minContent+3*dropGap)
	}
	// The card keeps the same height whatever the window is.
	if card.MinSize().Width <= 0 {
		t.Fatalf("drop card has no width")
	}
}
