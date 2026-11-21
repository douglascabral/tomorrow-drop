//go:build !darwin && !android && !ios

package main

// clipboardPaths reports no copied files on this platform yet.
func clipboardPaths() []string {
	return nil
}

// clipboardImage reports no picture: on this platform the clipboard is read as
// text only.
func clipboardImage() ([]byte, string, bool) {
	return nil, "", false
}

// clipboardContents reports no files on this platform yet.
func clipboardContents() []sharedFile {
	return nil
}
