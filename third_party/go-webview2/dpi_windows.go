//go:build windows

package webview2

import (
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

const wmDPIChanged = 0x02E0

var dpiUser32 = windows.NewLazySystemDLL("user32.dll")
var getDPIForWindow = dpiUser32.NewProc("GetDpiForWindow")
var getDPIForSystem = dpiUser32.NewProc("GetDpiForSystem")
var adjustWindowRectForDPI = dpiUser32.NewProc("AdjustWindowRectExForDpi")

func scaleForDPI(value int32, dpi uint32) int32 {
	if dpi == 0 {
		dpi = 96
	}
	if value <= 0 {
		return value
	}
	return int32((int64(value)*int64(dpi) + 48) / 96)
}

func windowDPI(hwnd uintptr) uint32 {
	dpi, _, _ := getDPIForWindow.Call(hwnd)
	if dpi == 0 {
		return 96
	}
	return uint32(dpi)
}

func scalePointForDPI(point w32.Point, dpi uint32) w32.Point {
	return w32.Point{X: scaleForDPI(point.X, dpi), Y: scaleForDPI(point.Y, dpi)}
}

func (w *webview) applyDPIChange(rect *w32.Rect) {
	if rect == nil {
		return
	}
	// Windows supplies physical coordinates, including negative monitor origins.
	_, _, _ = w32.User32SetWindowPos.Call(w.hwnd, 0,
		uintptr(rect.Left), uintptr(rect.Top), uintptr(rect.Right-rect.Left), uintptr(rect.Bottom-rect.Top),
		w32.SWPNoZOrder|w32.SWPNoActivate)
	w.browser.Resize()
	_ = w.browser.NotifyParentWindowPositionChanged()
}

func adjustClientRectForDPI(rect *w32.Rect, style uint32, dpi uint32) {
	_, _, _ = adjustWindowRectForDPI.Call(uintptr(unsafe.Pointer(rect)), uintptr(style), 0, 0, uintptr(dpi))
}
