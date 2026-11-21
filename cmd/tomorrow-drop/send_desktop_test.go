package main

import (
	"testing"

	"fyne.io/fyne/v2"
)

// On desktop the send screen uses the whole window, so a maximized window is
// not left mostly empty; the list of files takes the height that is left.
func TestSendOnDesktopUsesTheWholeWindow(t *testing.T) {
	for _, textMode := range []bool{false, true} {
		g := newTestGUI(t)
		g.mobile = false
		g.showSend("", textMode)
		content, ok := g.win.Content().(*fyne.Container)
		if !ok {
			t.Fatalf("send screen content is %T", g.win.Content())
		}
		if isColumn(content) {
			t.Fatalf("send screen (text mode %v) is held to the centered column", textMode)
		}
	}
}

// Large windows keep a wide column, so a maximized window is used well.
func TestDesktopColumnIsWideEnoughForLargeWindows(t *testing.T) {
	if maxColumnWidth < 960 {
		t.Fatalf("maxColumnWidth = %d, want at least 960 for maximized windows", maxColumnWidth)
	}
}
