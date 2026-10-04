//go:build windows

package webview2

import (
	"testing"
	"unsafe"
)

func assertTopmost(t *testing.T, hwnd uintptr, enabled bool, rect windowRect) {
	t.Helper()
	if windowIsTopmost(hwnd) != enabled {
		style, _, err := topmostGetStyle.Call(hwnd, ^uintptr(19))
		t.Fatalf("topmost != %v, hwnd=%#x style=%#x err=%v", enabled, hwnd, style, err)
	}
	menu, _, _ := stateUser32.NewProc("GetSystemMenu").Call(hwnd, 0)
	flags, _, _ := stateUser32.NewProc("GetMenuState").Call(menu, topmostMenuID, 0)
	if flags == 0xffffffff || (flags&0x8 != 0) != enabled {
		t.Fatalf("menu flags = %#x", flags)
	}
	var got windowRect
	if ok, _, _ := minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&got))); ok == 0 || got != rect {
		t.Fatalf("geometry changed: %+v -> %+v", rect, got)
	}
	if foreground, _, _ := stateUser32.NewProc("GetForegroundWindow").Call(); foreground == hwnd {
		t.Fatal("window activated")
	}
	if visible, _, _ := stateUser32.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
		t.Fatal("hidden window was shown")
	}
}

func TestAlwaysOnTopNativeToggleAndCleanup(t *testing.T) {
	stateTestThread(t)
	for _, initial := range []bool{false, true} {
		hwnd := stateTestWindow(t)
		var rect windowRect
		minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
		if err := installAlwaysOnTop(hwnd, initial); err != nil {
			t.Fatal(err)
		}
		assertTopmost(t, hwnd, initial, rect)
		for _, enabled := range []bool{!initial, initial} {
			stateUser32.NewProc("SendMessageW").Call(hwnd, 0x112, topmostMenuID|3, 0)
			assertTopmost(t, hwnd, enabled, rect)
		}
		// Refresh the menu if another native operation changes the OS state.
		if err := setWindowTopmost(hwnd, !initial); err != nil {
			t.Fatal(err)
		}
		stateUser32.NewProc("SendMessageW").Call(hwnd, 0x117, 0, 0)
		assertTopmost(t, hwnd, !initial, rect)
		stateUser32.NewProc("DestroyWindow").Call(hwnd)
		if exists, _, _ := stateUser32.NewProc("IsWindow").Call(hwnd); exists != 0 {
			t.Fatal("destroy retained window")
		}
		// User toggles are deliberately not persisted into the next window.
		reopened := stateTestWindow(t)
		if err := installAlwaysOnTop(reopened, initial); err != nil {
			t.Fatal(err)
		}
		if windowIsTopmost(reopened) != initial {
			t.Fatal("reopened state differs from configuration")
		}
	}
}

func TestAlwaysOnTopInvalidWindow(t *testing.T) {
	stateTestThread(t)
	if err := installAlwaysOnTop(0, true); err == nil {
		t.Fatal("invalid HWND accepted")
	}
	if err := setWindowTopmost(0, true); err == nil {
		t.Fatal("invalid HWND update accepted")
	}
}
