//go:build !ios && !android

package main

import "os"

// devicePlatform returns the computer name on desktop systems.
func devicePlatform() (string, string) {
	host, err := os.Hostname()
	if err != nil {
		return "", ""
	}
	return host, ""
}
