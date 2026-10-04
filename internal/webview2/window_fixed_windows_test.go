//go:build windows

package webview2

import (
	"testing"
	"unsafe"
)

type fixedTestView struct {
	fakeWebView
	hwnd uintptr
}

func (v *fixedTestView) Window() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&v.hwnd))
}

func TestFixedWindowNativeStyleCommandsAndIPC(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	var before, after windowRect
	minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&before)))
	style, _, _ := fixedGetStyle.Call(hwnd, ^uintptr(15))
	if err := installFixedWindow(hwnd, false); err != nil {
		t.Fatal(err)
	}
	unchanged, _, _ := fixedGetStyle.Call(hwnd, ^uintptr(15))
	if unchanged != style {
		t.Fatal("default style changed")
	}
	if err := installFixedWindow(hwnd, true); err != nil {
		t.Fatal(err)
	}
	fixed, _, _ := fixedGetStyle.Call(hwnd, ^uintptr(15))
	if fixed != style&^uintptr(0x50000) || fixed&0x20000 == 0 {
		t.Fatalf("styles: %#x -> %#x", style, fixed)
	}
	for _, command := range []uintptr{0xf030, 0xf003} {
		stateUser32.NewProc("SendMessageW").Call(hwnd, 0x112, command, 0)
	}
	if zoomed, _, _ := isZoomed.Call(hwnd); zoomed != 0 {
		t.Fatal("fixed window maximized")
	}
	minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&after)))
	if before != after {
		t.Fatal("geometry changed", before, after)
	}
	if visible, _, _ := stateUser32.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
		t.Fatal("window shown")
	}
	if foreground, _, _ := stateUser32.NewProc("GetForegroundWindow").Call(); foreground == hwnd {
		t.Fatal("window activated")
	}
	w := nativeWindow{view: &fixedTestView{hwnd: hwnd}, runtime: &Runtime{fixedSize: true}}
	if err := w.Maximize(); err == nil {
		t.Fatal("IPC bypassed fixed-size policy")
	}
	if state, err := w.State(); err != nil || state != "normal" {
		t.Fatal(state, err)
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if exists, _, _ := isNativeWindow.Call(hwnd); exists != 0 {
		t.Fatal("window not destroyed")
	}
	if err := installFixedWindow(0, true); err == nil {
		t.Fatal("invalid handle accepted")
	}
}

func TestFixedWindowRestoresPositionNotSavedSizeOrMaximize(t *testing.T) {
	stateTestThread(t)
	profile := t.TempDir()
	s := sampleWindowState()
	if err := saveWindowState(profile, s); err != nil {
		t.Fatal(err)
	}
	hwnd := stateTestWindow(t)
	if err := installFixedWindow(hwnd, true); err != nil {
		t.Fatal(err)
	}
	if err := installWindowState(hwnd, profile, s.AppID, 480, 360); err != nil {
		t.Fatal(err)
	}
	windowStates.Lock()
	item := windowStates.items[hwnd]
	windowStates.Unlock()
	got, err := captureWindowState(hwnd, item)
	if err != nil || got.Maximized || got.Normal != s.fitSize(got.Work, got.DPI, 480, 360) {
		t.Fatal(got, err)
	}
	if visible, _, _ := stateUser32.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
		t.Fatal("saved max state showed window")
	}
}
