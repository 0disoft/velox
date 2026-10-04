//go:build windows

package webview2

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNativeWindowTitleReadback(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	for _, title := range []string{"\uD55C\uAE00.md - Velox", "* note.md", ""} {
		if err := setNativeWindowTitle(hwnd, title); err != nil {
			t.Fatal(err)
		}
		buffer := make([]uint16, 513)
		count, _, _ := user32Window.NewProc("GetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
		if got := windows.UTF16ToString(buffer[:count]); got != title {
			t.Fatalf("native title = %q, want %q", got, title)
		}
	}
	user32Window.NewProc("DestroyWindow").Call(hwnd)
	if err := setNativeWindowTitle(hwnd, "closed"); err == nil {
		t.Fatal("destroyed window accepted title")
	}
	if err := (nativeWindow{}).SetTitle("missing"); err == nil {
		t.Fatal("missing window accepted title")
	}
}
