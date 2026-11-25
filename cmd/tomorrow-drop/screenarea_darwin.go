//go:build darwin && !ios

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

// tdScreenArea gives the size, in points, of the main screen without the menu
// bar and the Dock. It returns 0 when there is no screen.
static int tdScreenArea(double *w, double *h) {
	@autoreleasepool {
		NSScreen *s = [NSScreen mainScreen];
		if (s == nil) {
			return 0;
		}
		NSRect f = [s visibleFrame];
		*w = f.size.width;
		*h = f.size.height;
		return 1;
	}
}
*/
import "C"

import "fyne.io/fyne/v2"

// screenArea is the free area of the main screen, in Fyne units (points).
func screenArea() (fyne.Size, bool) {
	var w, h C.double
	if C.tdScreenArea(&w, &h) == 0 || w <= 0 || h <= 0 {
		return fyne.Size{}, false
	}
	return fyne.NewSize(float32(w), float32(h)), true
}
