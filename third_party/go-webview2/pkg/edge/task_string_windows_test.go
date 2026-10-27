//go:build windows

package edge

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var duplicateTaskString = windows.NewLazySystemDLL("shlwapi.dll").NewProc("SHStrDupW")

// SHStrDupW returns a typed output pointer allocated by CoTaskMemAlloc, matching
// the real COM getters and their CoTaskMemFree ownership contract.
func assignTaskString(out **uint16, value string) uintptr {
	*out = nil
	units, err := windows.UTF16FromString(value)
	if err != nil {
		return 0x80070057
	}
	hr, _, _ := duplicateTaskString.Call(uintptr(unsafe.Pointer(&units[0])), uintptr(unsafe.Pointer(out)))
	return hr
}
