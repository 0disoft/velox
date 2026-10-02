//go:build windows

package singleinstance

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const subclassID = 0x1f12

var user32 = windows.NewLazySystemDLL("user32.dll")
var comctl32 = windows.NewLazySystemDLL("comctl32.dll")
var registerMessage = user32.NewProc("RegisterWindowMessageW")
var setProp = user32.NewProc("SetPropW")
var getProp = user32.NewProc("GetPropW")
var removeProp = user32.NewProc("RemovePropW")
var enumWindows = user32.NewProc("EnumWindows")
var postMessage = user32.NewProc("PostMessageW")
var isIconic = user32.NewProc("IsIconic")
var showWindow = user32.NewProc("ShowWindow")
var setForeground = user32.NewProc("SetForegroundWindow")
var flashWindow = user32.NewProc("FlashWindowEx")
var setSubclass = comctl32.NewProc("SetWindowSubclass")
var removeSubclass = comctl32.NewProc("RemoveWindowSubclass")
var defSubclass = comctl32.NewProc("DefSubclassProc")
var callbacksOnce sync.Once
var subclassCallback, enumCallback, activationMessage uintptr
var initError error
var owners = struct {
	sync.Mutex
	items map[uintptr]*uint16
}{}

func initialize() error {
	callbacksOnce.Do(func() {
		name, _ := windows.UTF16PtrFromString("Velox.SingleInstance.Activate.v1")
		var err error
		activationMessage, _, err = registerMessage.Call(uintptr(unsafe.Pointer(name)))
		if activationMessage == 0 {
			initError = fmt.Errorf("register single-instance activation: %v", err)
			return
		}
		subclassCallback = windows.NewCallback(windowProc)
		enumCallback = windows.NewCallback(findWindow)
		owners.items = make(map[uintptr]*uint16)
	})
	return initError
}

// Attach must run on the window's UI thread, before publishing it to duplicates.
func (g *Guard) Attach(hwnd uintptr) error {
	if !g.primary || hwnd == 0 {
		return errors.New("single-instance window requires a primary instance")
	}
	if err := initialize(); err != nil {
		return err
	}
	property, _ := windows.UTF16PtrFromString(g.key)
	owners.Lock()
	owners.items[hwnd] = property
	owners.Unlock()
	result, _, err := setSubclass.Call(hwnd, subclassCallback, subclassID, 0)
	if result != 0 {
		result, _, err = setProp.Call(hwnd, uintptr(unsafe.Pointer(property)), 1)
	}
	if result == 0 {
		removeSubclass.Call(hwnd, subclassCallback, subclassID)
		owners.Lock()
		delete(owners.items, hwnd)
		owners.Unlock()
		return fmt.Errorf("attach single-instance window: %v", err)
	}
	return nil
}

var enumeration sync.Mutex
var searchedProperty *uint16

// Activate sends no data and never waits for the primary's initialization or UI.
func (g *Guard) Activate() {
	if initialize() != nil {
		return
	}
	property, _ := windows.UTF16PtrFromString(g.key)
	enumeration.Lock()
	defer enumeration.Unlock()
	searchedProperty = property
	enumWindows.Call(enumCallback, 0)
	searchedProperty = nil
}

func findWindow(hwnd, reference uintptr) uintptr {
	value, _, _ := getProp.Call(hwnd, uintptr(unsafe.Pointer(searchedProperty)))
	if value == 1 {
		postMessage.Call(hwnd, activationMessage, 0, 0)
		return 0
	}
	return 1
}

type flashInfo struct {
	Size                  uint32
	Window                uintptr
	Flags, Count, Timeout uint32
}

func windowProc(hwnd, message, wparam, lparam, id, reference uintptr) uintptr {
	if message == activationMessage && wparam == 0 && lparam == 0 {
		showWindow.Call(hwnd, 8) // SW_SHOWNA: also reveal a window hidden through its tray.
		minimized, _, _ := isIconic.Call(hwnd)
		if minimized != 0 {
			showWindow.Call(hwnd, 9) // SW_RESTORE
		}
		foreground, _, _ := setForeground.Call(hwnd)
		if foreground == 0 {
			info := flashInfo{Size: uint32(unsafe.Sizeof(flashInfo{})), Window: hwnd, Flags: 2, Count: 3}
			flashWindow.Call(uintptr(unsafe.Pointer(&info)))
		}
		return 0
	}
	if message == 0x82 { // WM_NCDESTROY
		owners.Lock()
		property := owners.items[hwnd]
		delete(owners.items, hwnd)
		owners.Unlock()
		if property != nil {
			removeProp.Call(hwnd, uintptr(unsafe.Pointer(property)))
		}
		removeSubclass.Call(hwnd, subclassCallback, subclassID)
	}
	result, _, _ := defSubclass.Call(hwnd, message, wparam, lparam)
	return result
}
