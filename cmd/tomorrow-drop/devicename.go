package main

import "tomorrow-drop/internal/ui"

// announcedName is the name this device announces on the network: the device
// name and its system where the platform gives them (see devicePlatform), or
// the computer name otherwise.
func announcedName() string {
	name, system := devicePlatform()
	if label := ui.DeviceLabel(name, system); label != "" {
		return label
	}
	return "tomorrow-drop"
}
