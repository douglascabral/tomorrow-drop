package main

import "fyne.io/fyne/v2"

// defaultWindowSize is the size of a phone-shaped window. It is the size on
// phones, where the system decides anyway, and the fallback on desktops whose
// screen area is not known.
var defaultWindowSize = fyne.NewSize(420, 640)

// The parts of the free screen area that the desktop window opens with: wide,
// and almost as tall as the screen, so the lists have room.
const (
	desktopWidthShare  = 0.8
	desktopHeightShare = 0.95
)

// initialWindowSize is the size the window opens with. On a desktop it is 80%
// of the free screen width and 95% of its height (area, without menu bar and dock), never smaller than
// the default size. ok is false when the platform could not tell the area.
func initialWindowSize(mobile bool, area fyne.Size, ok bool) fyne.Size {
	if mobile || !ok {
		return defaultWindowSize
	}
	w := area.Width * desktopWidthShare
	h := area.Height * desktopHeightShare
	if w < defaultWindowSize.Width {
		w = defaultWindowSize.Width
	}
	if h < defaultWindowSize.Height {
		h = defaultWindowSize.Height
	}
	return fyne.NewSize(w, h)
}
