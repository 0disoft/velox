//go:build windows

package clipboard

import (
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type readAPI struct {
	valid, open func(uintptr) bool
	available   func() bool
	close       func() bool
	get         func() uintptr
	size, lock  func(uintptr) uintptr
	copy        func([]uint16, uintptr)
	unlock      func(uintptr)
}

func NewWindowsReader(hwnd uintptr, dispatch func(func()), active func() bool, generation func() uint64, appName string) *Reader {
	api := newReadAPI()
	messageBox := windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")
	return &Reader{
		Dispatch: dispatch, Active: active, Generation: generation,
		Confirm: func() (bool, error) {
			if !api.valid(hwnd) {
				return false, ErrNative
			}
			body, err := windows.UTF16PtrFromString("Allow " + appName + " to read the current clipboard text?\n\nIt may contain private information. This approval applies only to this request.")
			if err != nil {
				return false, ErrNative
			}
			caption, _ := windows.UTF16PtrFromString("Read clipboard text")
			result, _, _ := messageBox.Call(hwnd, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), 0x124) // YESNO, QUESTION, default NO
			if result == 0 {
				return false, ErrNative
			}
			return result == 6, nil
		},
		Read: func() (string, error) { return readText(hwnd, api) },
	}
}

func newReadAPI() readAPI {
	user32 := windows.NewLazySystemDLL("user32.dll")
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	valid, open := user32.NewProc("IsWindow"), user32.NewProc("OpenClipboard")
	available, get := user32.NewProc("IsClipboardFormatAvailable"), user32.NewProc("GetClipboardData")
	close := user32.NewProc("CloseClipboard")
	size, lock, unlock := kernel32.NewProc("GlobalSize"), kernel32.NewProc("GlobalLock"), kernel32.NewProc("GlobalUnlock")
	move := windows.NewLazySystemDLL("ntdll.dll").NewProc("RtlMoveMemory")
	return readAPI{
		valid:     func(hwnd uintptr) bool { result, _, _ := valid.Call(hwnd); return result != 0 },
		open:      func(hwnd uintptr) bool { result, _, _ := open.Call(hwnd); return result != 0 },
		available: func() bool { result, _, _ := available.Call(13); return result != 0 },
		close:     func() bool { result, _, _ := close.Call(); return result != 0 },
		get:       func() uintptr { result, _, _ := get.Call(13); return result },
		size:      func(handle uintptr) uintptr { result, _, _ := size.Call(handle); return result },
		lock:      func(handle uintptr) uintptr { result, _, _ := lock.Call(handle); return result },
		copy: func(destination []uint16, address uintptr) {
			move.Call(uintptr(unsafe.Pointer(&destination[0])), address, uintptr(len(destination)*2))
		},
		unlock: func(handle uintptr) { unlock.Call(handle) },
	}
}

func readText(hwnd uintptr, api readAPI) (text string, err error) {
	if hwnd == 0 || !api.valid(hwnd) {
		return "", ErrNative
	}
	if !api.open(hwnd) {
		return "", ErrBusy
	}
	defer func() {
		if !api.close() && err == nil {
			text, err = "", ErrNative
		}
	}()
	if !api.available() {
		return "", ErrUnsupported
	}
	handle := api.get()
	if handle == 0 {
		return "", ErrNative
	}
	units := api.size(handle) / 2
	if units == 0 {
		return "", ErrInvalidText
	}
	// GlobalSize can include allocation padding; never copy an unbounded block.
	length := min(units, uintptr(MaxTextBytes+1))
	address := api.lock(handle)
	if address == 0 {
		return "", ErrNative
	}
	defer api.unlock(handle)
	buffer := make([]uint16, int(length))
	api.copy(buffer, address)
	for end, unit := range buffer {
		if unit != 0 {
			continue
		}
		for index := 0; index < end; index++ {
			unit := buffer[index]
			if unit >= 0xd800 && unit <= 0xdbff {
				if index+1 == end || buffer[index+1] < 0xdc00 || buffer[index+1] > 0xdfff {
					return "", ErrInvalidText
				}
				index++
			} else if unit >= 0xdc00 && unit <= 0xdfff {
				return "", ErrInvalidText
			}
		}
		text = string(utf16.Decode(buffer[:end]))
		if err := Validate(text); err != nil {
			return "", err
		}
		return text, nil
	}
	if units > uintptr(MaxTextBytes) {
		return "", ErrTooLarge
	}
	return "", ErrInvalidText
}
