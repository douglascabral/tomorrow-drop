//go:build !android

package main

// readShare reports nothing shared: receiving shares from other apps is done on
// Android only for now (see docs/MOBILE.md).
func readShare() (shareContent, bool) {
	return shareContent{}, false
}
