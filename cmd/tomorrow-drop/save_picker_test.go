package main

import "testing"

// Regression: saving a received file on a phone opens the system's storage
// picker, which takes the app to the background. Leaving the foreground then
// cancelled the reception, and the sender got its connection closed before the
// result. While the picker is open, the reception goes on.
func TestPickerKeepsTheReceptionRunning(t *testing.T) {
	g := newTestGUI(t)
	g.tlsPort = 0
	g.discPort = 0
	g.mobile = true

	g.setAvailable(true)
	ctx, done := g.beginOperation()
	defer done()
	g.pickingFile.Store(true)
	g.setAvailable(false)

	if ctx.Err() != nil {
		t.Fatalf("the system's save picker cancelled the reception")
	}
	g.pickingFile.Store(false)
}
