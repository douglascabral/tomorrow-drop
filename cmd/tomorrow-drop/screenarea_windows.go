//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
)

// screenArea is the work area of the primary monitor (without the taskbar), in
// Fyne units: the pixels of the system call divided by the display scale.
func screenArea() (fyne.Size, bool) {
	user32 := syscall.NewLazyDLL("user32.dll")
	var r struct{ left, top, right, bottom int32 }
	const spiGetWorkArea = 0x0030
	if ok, _, _ := user32.NewProc("SystemParametersInfoW").Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0); ok == 0 {
		return fyne.Size{}, false
	}
	scale := float32(1)
	if dpi, _, _ := user32.NewProc("GetDpiForSystem").Call(); dpi != 0 {
		scale = float32(dpi) / 96
	}
	w, h := float32(r.right-r.left)/scale, float32(r.bottom-r.top)/scale
	if w <= 0 || h <= 0 {
		return fyne.Size{}, false
	}
	return fyne.NewSize(w, h), true
}
