//go:build windows

package webview2

import (
	"errors"
	"runtime"
	"unsafe"
)

var flashWindowEx = user32Window.NewProc("FlashWindowEx")
var foregroundWindow = user32Window.NewProc("GetForegroundWindow")

type windowFlashInfo struct {
	Size    uint32
	Window  uintptr
	Flags   uint32
	Count   uint32
	Timeout uint32
}

func (w nativeWindow) RequestAttention(count uint32) error {
	handle, err := w.handle()
	if err != nil {
		return err
	}
	return setNativeWindowAttention(handle, count)
}

func (w nativeWindow) CancelAttention() error {
	handle, err := w.handle()
	if err != nil {
		return err
	}
	return setNativeWindowAttention(handle, 0)
}

func attentionInfo(handle uintptr, count uint32) windowFlashInfo {
	info := windowFlashInfo{Window: handle, Count: count}
	info.Size = uint32(unsafe.Sizeof(info))
	if count != 0 {
		info.Flags = 2
	} // FLASHW_TRAY only; no continuous timer flags.
	return info
}

func setNativeWindowAttention(handle uintptr, count uint32) error {
	if valid, _, _ := isNativeWindow.Call(handle); valid == 0 {
		return errors.New("native window is unavailable")
	}
	if count != 0 {
		if foreground, _, _ := foregroundWindow.Call(); foreground == handle {
			return nil
		}
	}
	info := attentionInfo(handle, count)
	// The BOOL reports the previous active state, not success or failure.
	flashWindowEx.Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(&info)
	return nil
}
