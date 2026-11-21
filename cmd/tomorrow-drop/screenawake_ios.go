//go:build ios

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework UIKit
#import <UIKit/UIKit.h>

// tdIdleTimer turns off (disabled=1) or on (disabled=0) the automatic screen
// lock. It runs on the main queue, as UIKit requires.
static void tdIdleTimer(int disabled) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[UIApplication sharedApplication].idleTimerDisabled = disabled != 0;
	});
}
*/
import "C"

import "fyne.io/fyne/v2"

// setScreenAwake keeps the iPhone or iPad screen on (true) or lets it lock (false).
func setScreenAwake(_ fyne.Window, on bool) {
	flag := C.int(0)
	if on {
		flag = 1
	}
	C.tdIdleTimer(flag)
}
