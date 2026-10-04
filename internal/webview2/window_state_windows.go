//go:build windows

package webview2

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowStateSubclassID = 0x1f11

var stateUser32 = user32Window
var stateComctl32 = permissionComctl32
var stateGetMonitorInfo = stateUser32.NewProc("GetMonitorInfoW")
var stateGetDPI = stateUser32.NewProc("GetDpiForWindow")
var stateSetSubclass = stateComctl32.NewProc("SetWindowSubclass")
var stateRemoveSubclass = stateComctl32.NewProc("RemoveWindowSubclass")
var stateDefSubclass = stateComctl32.NewProc("DefSubclassProc")
var stateMonitorFromRect = stateUser32.NewProc("MonitorFromRect")
var stateMonitorFromWindow = stateUser32.NewProc("MonitorFromWindow")
var stateSetWindowPos = stateUser32.NewProc("SetWindowPos")
var stateShowWindow = stateUser32.NewProc("ShowWindow")
var stateGetPlacement = stateUser32.NewProc("GetWindowPlacement")
var stateIsIconic = stateUser32.NewProc("IsIconic")
var stateSubclass uintptr
var stateSubclassOnce sync.Once
var windowStates = struct {
	sync.Mutex
	items map[uintptr]*windowStateOwner
}{}

type windowStateOwner struct {
	profile, appID string
	wasMaximized   bool
}

type nativePlacement struct {
	Length, Flags, Show    uint32
	MinX, MinY, MaxX, MaxY int32
	Normal                 windowRect
}

type nativeMonitorInfo struct {
	Size          uint32
	Monitor, Work windowRect
	Flags         uint32
}

func monitorInfo(monitor uintptr) (nativeMonitorInfo, error) {
	info := nativeMonitorInfo{Size: uint32(unsafe.Sizeof(nativeMonitorInfo{}))}
	result, _, err := stateGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info)))
	if result == 0 || !info.Work.valid() {
		return nativeMonitorInfo{}, fmt.Errorf("read monitor work area: %v", err)
	}
	return info, nil
}

func stateWindowDPI(hwnd uintptr) uint32 {
	value, _, _ := stateGetDPI.Call(hwnd)
	if value < 48 || value > 768 {
		return 96
	}
	return uint32(value)
}

func installWindowState(hwnd uintptr, profile, appID string, fixedWidth, fixedHeight uint) error {
	stateSubclassOnce.Do(func() {
		stateSubclass = windows.NewCallback(windowStateProc)
		windowStates.items = make(map[uintptr]*windowStateOwner)
	})
	item := &windowStateOwner{profile: profile, appID: appID}
	windowStates.Lock()
	windowStates.items[hwnd] = item
	windowStates.Unlock()
	result, _, err := stateSetSubclass.Call(hwnd, stateSubclass, windowStateSubclassID, 0)
	if result == 0 {
		windowStates.Lock()
		delete(windowStates.items, hwnd)
		windowStates.Unlock()
		return fmt.Errorf("install window state handler: %v", err)
	}
	if state, err := loadWindowState(profile, appID); err == nil {
		if err := restoreWindowStateSized(hwnd, state, fixedWidth, fixedHeight); err != nil {
			fmt.Fprintln(os.Stderr, "velox-host: saved window placement could not be fully restored")
		}
	}
	return nil
}

func restoreWindowState(hwnd uintptr, state windowState) error {
	return restoreWindowStateSized(hwnd, state, 0, 0)
}

func restoreWindowStateSized(hwnd uintptr, state windowState, fixedWidth, fixedHeight uint) error {
	monitor, _, _ := stateMonitorFromRect.Call(uintptr(unsafe.Pointer(&state.Normal)), 2)
	info, err := monitorInfo(monitor)
	if err != nil {
		return err
	}
	// First locate the window on the selected display so GetDpiForWindow sees its DPI.
	move := func(rect windowRect) error {
		result, _, err := stateSetWindowPos.Call(hwnd, 0, uintptr(rect.Left), uintptr(rect.Top), uintptr(rect.Right-rect.Left), uintptr(rect.Bottom-rect.Top), 0x14)
		if result == 0 {
			return fmt.Errorf("restore window rectangle: %v", err)
		}
		return nil
	}
	if err := move(state.fitSize(info.Work, state.DPI, fixedWidth, fixedHeight)); err != nil {
		return err
	}
	if err := move(state.fitSize(info.Work, stateWindowDPI(hwnd), fixedWidth, fixedHeight)); err != nil {
		return err
	}
	if state.Maximized && fixedWidth == 0 && fixedHeight == 0 {
		stateShowWindow.Call(hwnd, showMaximized)
	}
	return nil
}

func captureWindowState(hwnd uintptr, item *windowStateOwner) (windowState, error) {
	placement := nativePlacement{Length: uint32(unsafe.Sizeof(nativePlacement{}))}
	result, _, err := stateGetPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
	if result == 0 {
		return windowState{}, fmt.Errorf("read window placement: %v", err)
	}
	monitor, _, _ := stateMonitorFromWindow.Call(hwnd, 2)
	info, err := monitorInfo(monitor)
	if err != nil {
		return windowState{}, err
	}
	// WINDOWPLACEMENT uses workspace coordinates; SetWindowPos uses screen coordinates.
	normal := placement.Normal
	dx, dy := info.Work.Left-info.Monitor.Left, info.Work.Top-info.Monitor.Top
	normal.Left += dx
	normal.Right += dx
	normal.Top += dy
	normal.Bottom += dy
	minimized, _, _ := stateIsIconic.Call(hwnd)
	return windowState{Version: 1, AppID: item.appID, Normal: normal, Work: info.Work, DPI: stateWindowDPI(hwnd),
		Maximized: placement.Show == showMaximized || (minimized != 0 && item.wasMaximized)}, nil
}

func windowStateProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	windowStates.Lock()
	item := windowStates.items[hwnd]
	if message == 0x82 {
		delete(windowStates.items, hwnd)
	}
	windowStates.Unlock()
	if item != nil {
		switch message {
		case 0x5: // WM_SIZE: remember the pre-minimize state without disk writes.
			if wparam == 0 || wparam == 2 {
				item.wasMaximized = wparam == 2
			}
		case 0x2: // WM_DESTROY: accepted close only, while the HWND is still valid.
			state, err := captureWindowState(hwnd, item)
			if err == nil {
				err = saveWindowState(item.profile, state)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "velox-host: window state could not be saved")
			}
		case 0x82: // WM_NCDESTROY
			stateRemoveSubclass.Call(hwnd, stateSubclass, windowStateSubclassID)
		}
	}
	result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
	return result
}
