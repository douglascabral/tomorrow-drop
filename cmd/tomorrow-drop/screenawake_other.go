//go:build !android && !ios

package main

import "fyne.io/fyne/v2"

// setScreenAwake does nothing on desktop, where the screen is not turned off
// while the app is open.
func setScreenAwake(_ fyne.Window, _ bool) {}
