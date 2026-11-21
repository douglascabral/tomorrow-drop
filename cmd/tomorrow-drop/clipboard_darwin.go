//go:build darwin && !ios

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>

// tdClipboardPNG copies the picture on the general pasteboard as PNG. A picture
// that is only TIFF is converted. It returns 1 and the bytes (to be freed by the
// caller), or 0 when there is no picture.
// tdClipboardPaths returns the paths of the files copied on the general
// pasteboard (for example in the Finder), as C strings to be freed by the caller.
// It returns the number of files, or 0.
static int tdClipboardPaths(char ***paths) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		NSArray *urls = [pb readObjectsForClasses:@[[NSURL class]]
		                                  options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
		if (urls == nil || [urls count] == 0) {
			return 0;
		}
		NSUInteger n = [urls count];
		char **out = malloc(sizeof(char *) * n);
		if (out == NULL) {
			return 0;
		}
		for (NSUInteger i = 0; i < n; i++) {
			out[i] = strdup([[[urls objectAtIndex:i] path] UTF8String]);
		}
		*paths = out;
		return (int)n;
	}
}

static int tdClipboardPNG(void **out, size_t *n) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		NSData *data = [pb dataForType:NSPasteboardTypePNG];
		if (data == nil) {
			NSData *tiff = [pb dataForType:NSPasteboardTypeTIFF];
			if (tiff == nil) {
				return 0;
			}
			NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
			if (rep == nil) {
				return 0;
			}
			data = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
			if (data == nil) {
				return 0;
			}
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
// clipboardPaths returns the files copied on the clipboard, by their paths.
func clipboardPaths() []string {
	var list **C.char
	n := int(C.tdClipboardPaths(&list))
	if n == 0 {
		return nil
	}
	defer C.free(unsafe.Pointer(list))
	items := unsafe.Slice(list, n)
	paths := make([]string, 0, n)
	for _, p := range items {
		paths = append(paths, C.GoString(p))
		C.free(unsafe.Pointer(p))
	}
	return paths
}

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

// clipboardContents: on desktop copied files are read from their paths
// (clipboardPaths), so there are no contents to read.
func clipboardContents() []sharedFile {
	return nil
}
