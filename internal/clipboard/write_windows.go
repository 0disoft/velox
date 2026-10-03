//go:build windows

package clipboard

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type Writer struct {
	hwnd uintptr
	api  nativeAPI
}

func NewWindows(hwnd uintptr) *Writer { return &Writer{hwnd: hwnd, api: newNativeAPI()} }

type nativeAPI struct {
	valid, open, set func(uintptr) bool
	empty, close     func() bool
	alloc            func(uintptr) uintptr
	lock             func(uintptr) uintptr
	copy             func(uintptr, []uint16)
	unlock, free     func(uintptr)
}

func newNativeAPI() nativeAPI {
	user32 := windows.NewLazySystemDLL("user32.dll")
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	valid, open := user32.NewProc("IsWindow"), user32.NewProc("OpenClipboard")
	empty, close := user32.NewProc("EmptyClipboard"), user32.NewProc("CloseClipboard")
	set := user32.NewProc("SetClipboardData")
	alloc, lock := kernel32.NewProc("GlobalAlloc"), kernel32.NewProc("GlobalLock")
	unlock, free := kernel32.NewProc("GlobalUnlock"), kernel32.NewProc("GlobalFree")
	move := windows.NewLazySystemDLL("ntdll.dll").NewProc("RtlMoveMemory")
	return nativeAPI{
		valid: func(hwnd uintptr) bool { r, _, _ := valid.Call(hwnd); return r != 0 },
		open:  func(hwnd uintptr) bool { r, _, _ := open.Call(hwnd); return r != 0 },
		empty: func() bool { r, _, _ := empty.Call(); return r != 0 },
		close: func() bool { r, _, _ := close.Call(); return r != 0 },
		set:   func(handle uintptr) bool { r, _, _ := set.Call(13, handle); return r != 0 }, // CF_UNICODETEXT
		alloc: func(bytes uintptr) uintptr { r, _, _ := alloc.Call(0x42, bytes); return r }, // GMEM_MOVEABLE | GMEM_ZEROINIT
		lock:  func(handle uintptr) uintptr { r, _, _ := lock.Call(handle); return r },
		copy: func(destination uintptr, text []uint16) {
			move.Call(destination, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)*2))
		},
		unlock: func(handle uintptr) { unlock.Call(handle) },
		free:   func(handle uintptr) { free.Call(handle) },
	}
}

// WriteText runs on the host UI thread and never retries or reads the clipboard.
func (w *Writer) WriteText(text string) error { return writeText(w.hwnd, text, w.api) }

func writeText(hwnd uintptr, text string, api nativeAPI) (err error) {
	if err := Validate(text); err != nil {
		return err
	}
	if hwnd == 0 || !api.valid(hwnd) {
		return ErrNative
	}
	encoded, err := windows.UTF16FromString(text)
	if err != nil {
		return ErrInvalidText
	}
	// Prepare the entire movable allocation before replacing existing contents.
	handle := api.alloc(uintptr(len(encoded) * 2))
	if handle == 0 {
		return ErrNative
	}
	transferred := false
	defer func() {
		if !transferred {
			api.free(handle)
		}
	}()
	buffer := api.lock(handle)
	if buffer == 0 {
		return ErrNative
	}
	api.copy(buffer, encoded)
	api.unlock(handle)
	if !api.open(hwnd) {
		return ErrBusy
	}
	defer func() {
		if !api.close() && err == nil {
			err = ErrNative
		}
	}()
	if !api.empty() || !api.set(handle) {
		return ErrNative
	}
	// Windows owns the allocation after successful SetClipboardData.
	transferred = true
	return nil
}
