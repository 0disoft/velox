//go:build windows

package webview2

import (
	"testing"
	"unsafe"
)

func TestNativeMinimumTrackingRestoreAndCleanup(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	if err := installWindowMinimums(hwnd, 640, 480); err != nil {
		t.Fatal(err)
	}
	work, err := windowMinimumWork(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	limit := scaledWindowMinimum(640, 480, stateWindowDPI(hwnd), work)
	info := minimumTrackInfo{}
	user32Window.NewProc("SendMessageW").Call(hwnd, 0x24, 0, uintptr(unsafe.Pointer(&info)))
	if info.MinTrack.X < limit.X || info.MinTrack.Y < limit.Y || info.MinTrack.X > work.Right-work.Left || info.MinTrack.Y > work.Bottom-work.Top {
		t.Fatalf("native track = %+v, minimum %+v, work %+v", info, limit, work)
	}
	stateSetWindowPos.Call(hwnd, 0, 100, 100, 320, 240, 0x14)
	var rect windowRect
	minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if rect.Right-rect.Left < limit.X || rect.Bottom-rect.Top < limit.Y {
		t.Fatalf("restore below minimum: %+v", rect)
	}
	if visible, _, _ := user32Window.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
		t.Fatal("minimums showed hidden window")
	}
	if after, _, _ := foregroundWindow.Call(); after == hwnd {
		t.Fatal("minimums activated test window")
	}
	user32Window.NewProc("DestroyWindow").Call(hwnd)
	minimumOwners.Lock()
	item := minimumOwners.items[hwnd]
	minimumOwners.Unlock()
	if item != nil {
		t.Fatal("destroy retained minimum handler state")
	}
	if err := installWindowMinimums(0, 640, 480); err == nil {
		t.Fatal("invalid window accepted minimums")
	}
}

func TestNativeMinimumUnsetAndSingleAxis(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	if err := installWindowMinimums(hwnd, 0, 0); err != nil {
		t.Fatal(err)
	}
	minimumOwners.Lock()
	item := minimumOwners.items[hwnd]
	minimumOwners.Unlock()
	if item != nil {
		t.Fatal("unset limits attached a handler")
	}
	before := minimumTrackInfo{}
	user32Window.NewProc("SendMessageW").Call(hwnd, 0x24, 0, uintptr(unsafe.Pointer(&before)))
	if err := installWindowMinimums(hwnd, 640, 0); err != nil {
		t.Fatal(err)
	}
	after := minimumTrackInfo{}
	user32Window.NewProc("SendMessageW").Call(hwnd, 0x24, 0, uintptr(unsafe.Pointer(&after)))
	if after.MinTrack.Y != before.MinTrack.Y || after.MaxSize != before.MaxSize || after.MaxPosition != before.MaxPosition || after.MaxTrack != before.MaxTrack {
		t.Fatalf("minimums changed unset axis or maximums: before %+v after %+v", before, after)
	}
}

func TestNativeMinimumDPIChangeSuggestedRectangle(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	if err := installWindowMinimums(hwnd, 640, 480); err != nil {
		t.Fatal(err)
	}
	rect := windowRect{100, 100, 420, 340}
	monitor, _, _ := stateMonitorFromRect.Call(uintptr(unsafe.Pointer(&rect)), 2)
	info, err := monitorInfo(monitor)
	if err != nil {
		t.Fatal(err)
	}
	want := fitWindowMinimum(rect, info.Work, 640, 480, 144)
	user32Window.NewProc("SendMessageW").Call(hwnd, 0x2e0, 144|144<<16, uintptr(unsafe.Pointer(&rect)))
	if rect != want {
		t.Fatalf("DPI suggested rectangle = %+v, want %+v", rect, want)
	}
}
