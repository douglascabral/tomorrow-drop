//go:build !(darwin && !ios) && !windows

package main

import "fyne.io/fyne/v2"

// screenArea reports no area on this platform: the window keeps the default
// size (Linux has no common way to ask, and phones are sized by the system).
func screenArea() (fyne.Size, bool) {
	return fyne.Size{}, false
}
