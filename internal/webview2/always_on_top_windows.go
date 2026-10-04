//go:build windows

package webview2

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	topmostSubclassID = 0x1f16
	topmostMenuID     = 0x1f20 // System command IDs reserve the low four bits.
)

var topmostOnce sync.Once
var topmostCallback uintptr
var topmostGetStyle = stateUser32.NewProc("GetWindowLongPtrW")

func windowIsTopmost(hwnd uintptr) bool {
	style, _, _ := topmostGetStyle.Call(hwnd, ^uintptr(19)) // GWL_EXSTYLE = -20
	return style&0x8 != 0                                   // WS_EX_TOPMOST
}

func setWindowTopmost(hwnd uintptr, enabled bool) error {
	insertAfter := ^uintptr(1) // HWND_NOTOPMOST = -2
	if enabled {
		insertAfter = ^uintptr(0) // HWND_TOPMOST = -1
	}
	// Preserve geometry, visibility and focus; only change z-order.
	if ok, _, err := stateSetWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, 0x13); ok == 0 {
		return fmt.Errorf("set window topmost: %v", err)
	}
	return nil
}

func syncTopmostMenu(hwnd uintptr) {
	menu, _, _ := stateUser32.NewProc("GetSystemMenu").Call(hwnd, 0)
	flags := uintptr(0)
	if windowIsTopmost(hwnd) {
		flags = 0x8 // MF_CHECKED, by command ID.
	}
	stateUser32.NewProc("CheckMenuItem").Call(menu, topmostMenuID, flags)
}

func installAlwaysOnTop(hwnd uintptr, enabled bool) error {
	// Resolve before entering a native callback, including default-off windows.
	if err := stateSetWindowPos.Find(); err != nil {
		return fmt.Errorf("resolve topmost window API: %w", err)
	}
	topmostOnce.Do(func() { topmostCallback = windows.NewCallback(topmostWindowProc) })
	menu, _, _ := stateUser32.NewProc("GetSystemMenu").Call(hwnd, 0)
	if menu == 0 {
		return fmt.Errorf("get topmost system menu")
	}
	label, _ := windows.UTF16PtrFromString("Always on &top")
	if ok, _, _ := stateUser32.NewProc("AppendMenuW").Call(menu, 0, topmostMenuID, uintptr(unsafe.Pointer(label))); ok == 0 {
		return fmt.Errorf("add topmost menu")
	}
	rollback := func() {
		stateRemoveSubclass.Call(hwnd, topmostCallback, topmostSubclassID)
		stateUser32.NewProc("RemoveMenu").Call(menu, topmostMenuID, 0)
	}
	if ok, _, err := stateSetSubclass.Call(hwnd, topmostCallback, topmostSubclassID, 0); ok == 0 {
		rollback()
		return fmt.Errorf("install topmost window handler: %v", err)
	}
	if enabled {
		if err := setWindowTopmost(hwnd, true); err != nil {
			rollback()
			return err
		}
	}
	syncTopmostMenu(hwnd)
	return nil
}

func topmostWindowProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	if subclassID != topmostSubclassID || reference != 0 {
		result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
		return result
	}
	switch message {
	case 0x112: // WM_SYSCOMMAND
		if wparam&0xfff0 == topmostMenuID {
			if err := setWindowTopmost(hwnd, !windowIsTopmost(hwnd)); err != nil {
				fmt.Fprintln(os.Stderr, "velox-host:", err)
			}
			syncTopmostMenu(hwnd)
			return 0
		}
	case 0x117: // WM_INITMENUPOPUP: refresh from OS state, not a cached boolean.
		syncTopmostMenu(hwnd)
	case 0x82: // WM_NCDESTROY
		stateRemoveSubclass.Call(hwnd, topmostCallback, topmostSubclassID)
	}
	result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
	return result
}
