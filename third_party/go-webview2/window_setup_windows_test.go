//go:build windows

package webview2

import (
	"errors"
	"runtime"
	"testing"
)

func TestBeforeShowHookRunsBeforeVisibilityAndCleansFailure(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w := &webview{browser: &dpiBrowser{}}
	want := errors.New("window setup failure")
	var observed uintptr
	if w.CreateWithOptions(WindowOptions{BeforeShow: func(hwnd uintptr) error {
		observed = hwnd
		if visible, _, _ := dpiUser32.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
			t.Error("hook ran after ShowWindow")
		}
		if getWindowContext(hwnd) != w {
			t.Error("window context missing in hook")
		}
		return want
	}}) {
		t.Fatal("failed hook accepted")
	}
	if observed == 0 || w.windowSetupErr != want || w.hwnd != 0 {
		t.Fatal("failure was not retained", w)
	}
	if valid, _, _ := dpiUser32.NewProc("IsWindow").Call(observed); valid != 0 || getWindowContext(observed) != nil {
		t.Fatal("setup failure retained native window/context")
	}
}
