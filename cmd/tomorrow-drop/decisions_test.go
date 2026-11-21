package main

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/identity"
)

// Regression: the back arrow of a decision screen (pairing, a request to send
// files) went home and left the other device waiting for an answer. Going back
// is declining.
func TestBackOnADecisionScreenDeclines(t *testing.T) {
	g := newTestGUI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	answer := make(chan bool, 1)
	go func() { answer <- g.askScreen(ctx, "Pairing", widget.NewLabel("x"), "Yes", "No") }()
	waitForButton(t, g, "Yes")

	back := findButton(g.win.Content(), "")
	if back == nil {
		t.Fatalf("the decision screen has no back button")
	}
	back.OnTapped()
	select {
	case ok := <-answer:
		if ok {
			t.Fatalf("going back accepted the request")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("going back left the question unanswered")
	}
}

// Regression: "Cancel transfer" on the receiving screen did nothing.
func TestCancelOnTheReceivingScreenStopsTheReception(t *testing.T) {
	g := newTestGUI(t)
	ctx, done := g.beginOperation()
	defer done()
	g.beginReceiveMeter("laptop", 1, 100)
	waitForButton(t, g, g.tr.T("gui.send.cancel"))

	findButton(g.win.Content(), g.tr.T("gui.send.cancel")).OnTapped()
	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatalf("cancel did not stop the reception in progress")
	}
}

// A reception that is cancelled or cut does not leave the progress ring on
// screen: the person is told, and can wait for files again.
func TestInterruptedReceptionShowsAResult(t *testing.T) {
	g := newTestGUI(t)
	g.beginReceiveMeter("laptop", 1, 100)
	if !g.receiving.Load() {
		t.Fatalf("the application does not know a reception is in progress")
	}
	g.endReception(context.Canceled)
	waitForButton(t, g, g.tr.T("gui.done.again"))
	if g.receiving.Load() {
		t.Fatalf("the reception is still marked as in progress")
	}
}

// Enter in the address fields connects, and the card keeps its natural height.
func TestManualConnectionSubmitsWithEnter(t *testing.T) {
	g := newTestGUI(t)
	g.showManual()
	entries := findEntries(g.win.Content())
	if len(entries) != 2 {
		t.Fatalf("manual screen has %d fields, want 2", len(entries))
	}
	for _, e := range entries {
		if e.OnSubmitted == nil {
			t.Fatalf("a field of the manual screen does not submit with Enter")
		}
	}
}

// The name of a new device starts from the name it announces, when that is a
// name and not an address; otherwise from the start of its fingerprint.
func TestNameSuggestionUsesTheAnnouncedName(t *testing.T) {
	g := newTestGUI(t)
	id, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	fp := id.Fingerprint()

	g.pairHint = "MacBook-Air-de-Douglas"
	if got := g.nameSuggestion(fp); got != "MacBook-Air-de-Douglas" {
		t.Fatalf("suggestion = %q, want the announced name", got)
	}
	g.pairHint = "10.0.0.2:47521"
	if got := g.nameSuggestion(fp); got != fp.Hex()[:8] {
		t.Fatalf("suggestion for an address = %q, want the fingerprint start", got)
	}
	g.pairHint = ""
	if got := g.nameSuggestion(fp); got != fp.Hex()[:8] {
		t.Fatalf("suggestion with no name = %q, want the fingerprint start", got)
	}
}

// findEntries returns every text field under obj.
func findEntries(obj fyne.CanvasObject) []*widget.Entry {
	switch o := obj.(type) {
	case *widget.Entry:
		return []*widget.Entry{o}
	case *container.Scroll:
		return findEntries(o.Content)
	case *container.ThemeOverride:
		return findEntries(o.Content)
	case *fyne.Container:
		var out []*widget.Entry
		for _, c := range o.Objects {
			out = append(out, findEntries(c)...)
		}
		return out
	}
	return nil
}
