//go:build ios

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework UIKit -framework Foundation
#include <stdlib.h>
#include <string.h>
#import <UIKit/UIKit.h>

// tdClipboardPNG copies the picture on the general pasteboard as PNG. It returns
// 1 and the bytes (to be freed by the caller), or 0 when there is no picture.
static int tdClipboardPNG(void **out, size_t *n) {
	@autoreleasepool {
		UIImage *image = [[UIPasteboard generalPasteboard] image];
		if (image == nil) {
			return 0;
		}
		NSData *data = UIImagePNGRepresentation(image);
		if (data == nil) {
			return 0;
		}
		*n = [data length];
		*out = malloc(*n);
		if (*out == NULL) {
			return 0;
		}
		memcpy(*out, [data bytes], *n);
		return 1;
	}
}
*/
import "C"

import "unsafe"

// clipboardImage returns the picture on the clipboard, as PNG, when there is one.
func clipboardImage() ([]byte, string, bool) {
	var p unsafe.Pointer
	var n C.size_t
	if C.tdClipboardPNG(&p, &n) == 0 {
		return nil, "", false
	}
	defer C.free(p)
	return C.GoBytes(p, C.int(n)), ".png", true
}

// clipboardContents reports no files: iOS does not give other apps' files
// through the pasteboard.
func clipboardContents() []sharedFile {
	return nil
}

// clipboardPaths reports no paths on iOS: a copied file has no path other apps
// can read.
func clipboardPaths() []string {
	return nil
}
