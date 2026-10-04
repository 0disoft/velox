//go:build windows

package webview2

import (
	"testing"
	"unsafe"
)

func TestAttentionInfoHasNativeLayoutAndBoundedFlags(t *testing.T) {
	for _, count := range []uint32{0, 1, 3, 5} {
		info := attentionInfo(42, count)
		wantFlags := uint32(2)
		if count == 0 {
			wantFlags = 0
		}
		if info.Window != 42 || info.Flags != wantFlags || info.Count != count || info.Timeout != 0 || uintptr(info.Size) != unsafe.Sizeof(info) {
			t.Fatalf("flash info = %+v", info)
		}
		if unsafe.Sizeof(uintptr(0)) == 8 && (info.Size != 32 || unsafe.Offsetof(info.Window) != 8 || unsafe.Offsetof(info.Timeout) != 24) {
			t.Fatalf("incorrect Windows x64 FLASHWINFO layout: %+v", info)
		}
	}
}

func TestNativeAttentionDoesNotShowWindowOrChangeForeground(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	for _, count := range []uint32{1, 5, 0} {
		if err := setNativeWindowAttention(hwnd, count); err != nil {
			t.Fatal(err)
		}
		if visible, _, _ := user32Window.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
			t.Fatal("attention made the hidden test window visible")
		}
	}
	if after, _, _ := foregroundWindow.Call(); after == hwnd {
		t.Fatal("attention activated the hidden test window")
	}
	user32Window.NewProc("DestroyWindow").Call(hwnd)
	if err := setNativeWindowAttention(hwnd, 1); err == nil {
		t.Fatal("destroyed window accepted attention")
	}
	if err := setNativeWindowAttention(0, 0); err == nil {
		t.Fatal("missing window accepted cancel")
	}
	if err := (nativeWindow{}).RequestAttention(1); err == nil {
		t.Fatal("missing view accepted attention")
	}
	if err := (nativeWindow{}).CancelAttention(); err == nil {
		t.Fatal("missing view accepted cancel")
	}
}
