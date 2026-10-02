//go:build windows

package externalurl

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var messageBox = user32.NewProc("MessageBoxW")
var isWindow = user32.NewProc("IsWindow")

func NewWindows(hwnd uintptr, isClosing func() bool) *Opener {
	valid := func() bool { result, _, _ := isWindow.Call(hwnd); return result != 0 }
	return &Opener{
		IsClosing: isClosing,
		Confirm: func(target string) (bool, error) {
			if !valid() {
				return false, errors.New("external browser owner is unavailable")
			}
			body, _ := windows.UTF16PtrFromString("Open this HTTPS link in your default browser?\n\n" + target)
			caption, _ := windows.UTF16PtrFromString("Open external link")
			result, _, _ := messageBox.Call(hwnd, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), 0x124) // YESNO, QUESTION, default NO
			if result == 0 {
				return false, errors.New("external-link confirmation failed")
			}
			return result == 6, nil
		},
		Launch: func(target string) error {
			if !valid() {
				return errors.New("external browser owner is unavailable")
			}
			verb, _ := windows.UTF16PtrFromString("open")
			uri, _ := windows.UTF16PtrFromString(target)
			// Pass only the validated URI as the object; never add arguments or invoke a shell command.
			return windows.ShellExecute(windows.Handle(hwnd), verb, uri, nil, nil, windows.SW_SHOWNORMAL)
		},
	}
}

func NotifyWindowsFailure(hwnd uintptr) {
	valid, _, _ := isWindow.Call(hwnd)
	if valid == 0 {
		return
	}
	body, _ := windows.UTF16PtrFromString("The external link could not be opened in your default browser.")
	caption, _ := windows.UTF16PtrFromString("Open external link")
	messageBox.Call(hwnd, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), 0x10)
}
