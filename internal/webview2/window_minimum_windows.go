//go:build windows

package webview2

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const minimumSubclassID = 0x1f15

var minimumGetRect = user32Window.NewProc("GetWindowRect")
var minimumCallback uintptr
var minimumOnce sync.Once
var minimumOwners = struct {
	sync.Mutex
	items map[uintptr]*windowMinimumOwner
}{}

type windowMinimumOwner struct {
	width, height uint
	adjusting     bool
}

type minimumTrackInfo struct {
	Reserved, MaxSize, MaxPosition, MinTrack, MaxTrack minimumPoint
}

func installWindowMinimums(hwnd uintptr, width, height uint) error {
	if width == 0 && height == 0 {
		return nil
	}
	minimumOnce.Do(func() {
		minimumCallback = windows.NewCallback(windowMinimumProc)
		minimumOwners.items = make(map[uintptr]*windowMinimumOwner)
	})
	item := &windowMinimumOwner{width: width, height: height}
	minimumOwners.Lock()
	minimumOwners.items[hwnd] = item
	minimumOwners.Unlock()
	installed, _, err := stateSetSubclass.Call(hwnd, minimumCallback, minimumSubclassID, 0)
	if installed == 0 {
		minimumOwners.Lock()
		delete(minimumOwners.items, hwnd)
		minimumOwners.Unlock()
		return fmt.Errorf("install minimum window size handler: %v", err)
	}
	if err := enforceWindowMinimum(hwnd, item); err != nil {
		stateRemoveSubclass.Call(hwnd, minimumCallback, minimumSubclassID)
		minimumOwners.Lock()
		delete(minimumOwners.items, hwnd)
		minimumOwners.Unlock()
		return err
	}
	return nil
}

func windowMinimumWork(hwnd uintptr) (windowRect, error) {
	monitor, _, _ := stateMonitorFromWindow.Call(hwnd, 2)
	info, err := monitorInfo(monitor)
	return info.Work, err
}

func enforceWindowMinimum(hwnd uintptr, item *windowMinimumOwner) error {
	if item.adjusting {
		return nil
	}
	if iconic, _, _ := isIconic.Call(hwnd); iconic != 0 {
		return nil
	}
	if zoomed, _, _ := isZoomed.Call(hwnd); zoomed != 0 {
		return nil
	}
	var rect windowRect
	if ok, _, err := minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ok == 0 {
		return fmt.Errorf("read minimum window rectangle: %v", err)
	}
	work, err := windowMinimumWork(hwnd)
	if err != nil {
		return err
	}
	fitted := fitWindowMinimum(rect, work, item.width, item.height, stateWindowDPI(hwnd))
	if fitted == rect {
		return nil
	}
	// SetWindowPos may synchronously send another WM_SIZE on the UI thread.
	item.adjusting = true
	defer func() { item.adjusting = false }()
	ok, _, err := stateSetWindowPos.Call(hwnd, 0, uintptr(fitted.Left), uintptr(fitted.Top),
		uintptr(fitted.Right-fitted.Left), uintptr(fitted.Bottom-fitted.Top), 0x14)
	if ok == 0 {
		return fmt.Errorf("apply minimum window size: %v", err)
	}
	return nil
}

// Win32 owns this message data until the synchronous callback returns.
// Reinterpret the native address word only for messages with pointer LPARAMs.
func minimumMessageData[T any](address uintptr) *T {
	return (*T)(*(*unsafe.Pointer)(unsafe.Pointer(&address)))
}

func windowMinimumProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	minimumOwners.Lock()
	item := minimumOwners.items[hwnd]
	if message == 0x82 {
		delete(minimumOwners.items, hwnd)
	}
	minimumOwners.Unlock()
	if item != nil && message == 0x2e0 && lparam != 0 { // WM_DPICHANGED suggested normal rectangle.
		if zoomed, _, _ := isZoomed.Call(hwnd); zoomed == 0 {
			rect := minimumMessageData[windowRect](lparam)
			monitor, _, _ := stateMonitorFromRect.Call(lparam, 2)
			if info, err := monitorInfo(monitor); err == nil {
				*rect = fitWindowMinimum(*rect, info.Work, item.width, item.height, uint32(wparam&0xffff))
			}
		}
	}
	result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
	if item != nil {
		switch message {
		case 0x24: // WM_GETMINMAXINFO: retain normal OS maximum sizing.
			if lparam != 0 {
				if work, err := windowMinimumWork(hwnd); err == nil {
					info := minimumMessageData[minimumTrackInfo](lparam)
					limit := scaledWindowMinimum(item.width, item.height, stateWindowDPI(hwnd), work)
					if item.width != 0 {
						info.MinTrack.X = min(max(info.MinTrack.X, limit.X), work.Right-work.Left)
					}
					if item.height != 0 {
						info.MinTrack.Y = min(max(info.MinTrack.Y, limit.Y), work.Bottom-work.Top)
					}
				}
			}
		case 0x5: // WM_SIZE: also protect normal restore/programmatic sizing.
			if wparam == 0 {
				_ = enforceWindowMinimum(hwnd, item)
			}
		case 0x82: // WM_NCDESTROY
			stateRemoveSubclass.Call(hwnd, minimumCallback, minimumSubclassID)
		}
	}
	return result
}
