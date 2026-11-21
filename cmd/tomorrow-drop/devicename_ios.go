//go:build ios

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UIKit
#include <stdlib.h>
#import <UIKit/UIKit.h>

// tdDeviceName returns the device name set by its owner (UIDevice.name), or
// the model when iOS withholds the name. Caller frees the result.
static char *tdDeviceName(void) {
	@autoreleasepool {
		return strdup([[[UIDevice currentDevice] name] UTF8String]);
	}
}

// tdSystem returns the system name and version, for example "iPadOS 26.1".
static char *tdSystem(void) {
	@autoreleasepool {
		UIDevice *d = [UIDevice currentDevice];
		NSString *s = [NSString stringWithFormat:@"%@ %@", d.systemName, d.systemVersion];
		return strdup([s UTF8String]);
	}
}
*/
import "C"

import "unsafe"

// devicePlatform returns the name and the system of this iPhone or iPad.
func devicePlatform() (string, string) {
	name := C.tdDeviceName()
	defer C.free(unsafe.Pointer(name))
	system := C.tdSystem()
	defer C.free(unsafe.Pointer(system))
	return C.GoString(name), C.GoString(system)
}
