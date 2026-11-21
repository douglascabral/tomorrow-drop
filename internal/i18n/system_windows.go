//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

// SystemLocale reads the Windows user locale name (for example pt-BR).
func SystemLocale() string {
	buf := make([]uint16, 85)
	n, _, _ := syscall.NewLazyDLL("kernel32.dll").
		NewProc("GetUserDefaultLocaleName").
		Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
