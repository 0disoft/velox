//go:build windows

package webview2

import (
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
)

const fixedSubclassID = 0x1f17

var fixedGetStyle = user32Window.NewProc("GetWindowLongPtrW")
var fixedSetStyle = user32Window.NewProc("SetWindowLongPtrW")
var fixedCallback uintptr
var fixedOnce sync.Once

func installFixedWindow(hwnd uintptr, fixed bool) error {
	if !fixed {
		return nil
	}
	if valid, _, _ := isNativeWindow.Call(hwnd); valid == 0 {
		return fmt.Errorf("fixed window is unavailable")
	}
	// Resolve APIs before native message callbacks can reenter the handler.
	if err := stateSetWindowPos.Find(); err != nil {
		return err
	}
	fixedOnce.Do(func() { fixedCallback = windows.NewCallback(fixedWindowProc) })
	style, _, _ := fixedGetStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE = -16
	if ok, _, err := stateSetSubclass.Call(hwnd, fixedCallback, fixedSubclassID, 0); ok == 0 {
		return fmt.Errorf("install fixed window handler: %v", err)
	}
	newStyle := style &^ uintptr(0x40000|0x10000) // WS_THICKFRAME | WS_MAXIMIZEBOX
	if previous, _, err := fixedSetStyle.Call(hwnd, ^uintptr(15), newStyle); previous == 0 {
		stateRemoveSubclass.Call(hwnd, fixedCallback, fixedSubclassID)
		return fmt.Errorf("set fixed window style: %v", err)
	}
	if ok, _, err := stateSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, 0x37); ok == 0 {
		fixedSetStyle.Call(hwnd, ^uintptr(15), style)
		stateRemoveSubclass.Call(hwnd, fixedCallback, fixedSubclassID)
		return fmt.Errorf("apply fixed window frame: %v", err)
	}
	return nil
}

func fixedWindowProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	if message == 0x112 { // WM_SYSCOMMAND: also cover keyboard/system-menu commands.
		command := wparam & 0xfff0
		if command == 0xf000 || command == 0xf030 {
			return 0
		} // SC_SIZE / SC_MAXIMIZE
	}
	if message == 0x82 {
		stateRemoveSubclass.Call(hwnd, fixedCallback, fixedSubclassID)
	}
	result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
	return result
}
