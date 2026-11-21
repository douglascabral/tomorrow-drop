package main

import "testing"

// On a phone the screen stays on while the app is open and may turn off in the
// background, which is when transfers are interrupted (task 057).
func TestPhoneKeepsTheScreenOnWhileTheAppIsOpen(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = true
	var calls []bool
	g.screenOn = func(on bool) { calls = append(calls, on) }

	g.setAvailable(true)
	g.setAvailable(false)
	if len(calls) != 2 || calls[0] != true || calls[1] != false {
		t.Fatalf("screen-on calls = %v, want on in the foreground and off in the background", calls)
	}
}

// Desktop is not affected.
func TestDesktopDoesNotChangeScreenBlanking(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = false
	called := false
	g.screenOn = func(bool) { called = true }
	g.setAvailable(true)
	g.setAvailable(false)
	if called {
		t.Fatalf("desktop changed the screen blanking")
	}
}
