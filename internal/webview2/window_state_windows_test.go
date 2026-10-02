//go:build windows

package webview2

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func stateTestThread(t *testing.T) {
	t.Helper()
	runtime.LockOSThread()
	proc := stateUser32.NewProc("SetThreadDpiAwarenessContext")
	previous, _, _ := proc.Call(^uintptr(3)) // PER_MONITOR_AWARE_V2
	if previous == 0 {
		previous, _, _ = proc.Call(^uintptr(2)) // PER_MONITOR_AWARE
	}
	t.Cleanup(func() {
		if previous != 0 {
			proc.Call(previous)
		}
		runtime.UnlockOSThread()
	})
	if previous == 0 {
		t.Fatal("could not select per-monitor DPI for native state test")
	}
}

func stateTestWindow(t *testing.T) uintptr {
	t.Helper()
	class, _ := windows.UTF16PtrFromString("STATIC")
	title, _ := windows.UTF16PtrFromString("Velox window state test")
	hwnd, _, err := stateUser32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0xcf0000, 100, 100, 800, 600, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("create disposable native window: %v", err)
	}
	t.Cleanup(func() { stateUser32.NewProc("DestroyWindow").Call(hwnd) })
	return hwnd
}

func TestWindowStateNativeCaptureRestoreAndDestroy(t *testing.T) {
	stateTestThread(t)
	profile := t.TempDir()
	hwnd := stateTestWindow(t)
	if err := installWindowState(hwnd, profile, "dev.velox.state-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(profile, windowStateFile)); !os.IsNotExist(err) {
		t.Fatalf("install wrote state: %v", err)
	}
	windowStates.Lock()
	item := windowStates.items[hwnd]
	windowStates.Unlock()
	state, err := captureWindowState(hwnd, item)
	if err != nil || !state.valid(item.appID) {
		t.Fatalf("captured = %+v, %v", state, err)
	}
	state.Normal.Left += 20
	state.Normal.Right += 20
	state.Normal.Top += 30
	state.Normal.Bottom += 30
	if err := restoreWindowState(hwnd, state); err != nil {
		t.Fatal(err)
	}
	got, err := captureWindowState(hwnd, item)
	if err != nil || got.Normal != state.fit(got.Work, got.DPI) {
		t.Fatalf("restored = %+v, %v; source %+v", got, err, state)
	}
	// Moving/resizing events have not written anything; only actual destruction does.
	if _, err := os.Stat(filepath.Join(profile, windowStateFile)); !os.IsNotExist(err) {
		t.Fatalf("move wrote state: %v", err)
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	saved, err := loadWindowState(profile, item.appID)
	if err != nil || saved.Normal != got.Normal {
		t.Fatalf("destroy saved = %+v, %v", saved, err)
	}
	windowStates.Lock()
	remaining := windowStates.items[hwnd]
	windowStates.Unlock()
	if remaining != nil {
		t.Fatal("destroy retained window subclass owner")
	}
}

func TestWindowStateNativeMinimizedMaximizedReopensVisible(t *testing.T) {
	stateTestThread(t)
	profile := t.TempDir()
	hwnd := stateTestWindow(t)
	if err := installWindowState(hwnd, profile, "dev.velox.state-test"); err != nil {
		t.Fatal(err)
	}
	stateUser32.NewProc("ShowWindow").Call(hwnd, showMaximized)
	stateUser32.NewProc("ShowWindow").Call(hwnd, showMinimized)
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	state, err := loadWindowState(profile, "dev.velox.state-test")
	if err != nil || !state.Maximized {
		t.Fatalf("minimized-maximized state = %+v, %v", state, err)
	}
	reopened := stateTestWindow(t)
	if err := installWindowState(reopened, profile, state.AppID); err != nil {
		t.Fatal(err)
	}
	minimized, _, _ := stateUser32.NewProc("IsIconic").Call(reopened)
	maximized, _, _ := stateUser32.NewProc("IsZoomed").Call(reopened)
	if minimized != 0 || maximized == 0 {
		t.Fatalf("reopened minimized=%d maximized=%d", minimized, maximized)
	}
}
