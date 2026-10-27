//go:build windows

package webview2

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// LPARAM can contain either numbers or native addresses. Copy only the bounded
// message payload while the callback owns it; never turn it into a Go pointer.
// The Windows APIs also reject inaccessible or incomplete payloads.
func readWindowMessage(address uintptr, target unsafe.Pointer, size uintptr) bool {
	if address == 0 || target == nil || size == 0 {
		return false
	}
	var copied uintptr
	err := windows.ReadProcessMemory(windows.CurrentProcess(), address, (*byte)(target), size, &copied)
	return err == nil && copied == size
}

func writeWindowMessage(address uintptr, source unsafe.Pointer, size uintptr) bool {
	if address == 0 || source == nil || size == 0 {
		return false
	}
	var copied uintptr
	err := windows.WriteProcessMemory(windows.CurrentProcess(), address, (*byte)(source), size, &copied)
	return err == nil && copied == size
}
