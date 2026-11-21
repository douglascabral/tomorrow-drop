// Command tomorrow-drop-ios is the core of the native iPhone and iPad app. It is
// built as a C archive (buildmode=c-archive) and called from SwiftUI through the
// functions in bridge.go. It holds no user interface: it runs the same core
// packages as the other applications and reports state as JSON.
package main

import "C"

func main() {}
