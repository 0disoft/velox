//go:build windows

package webview2

import "golang.org/x/sys/windows"

// ID 11 is separate from the main icon so LR_SHARED cannot return its larger
// cached image for the title bar. Windows owns both shared handles.
func setSmallWindowIcon(hwnd uintptr) {
	var module windows.Handle
	if windows.GetModuleHandleEx(0, nil, &module) != nil {
		return
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	metrics := user32.NewProc("GetSystemMetrics")
	width, _, _ := metrics.Call(49)  // SM_CXSMICON
	height, _, _ := metrics.Call(50) // SM_CYSMICON
	icon, _, _ := user32.NewProc("LoadImageW").Call(uintptr(module), 11, 1, width, height, 0x8000)
	if icon != 0 {
		user32.NewProc("SendMessageW").Call(hwnd, 0x80, 0, icon) // WM_SETICON, ICON_SMALL
	}
}
