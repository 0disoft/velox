//go:build windows

package webview2

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

func TestDestroyAfterRunReleasesNativeWindow(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	user32 := windows.NewLazySystemDLL("user32.dll")
	var msg w32.Msg
	peek := user32.NewProc("PeekMessageW")
	peek.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		t.Fatal(err)
	}
	class, _ := windows.UTF16PtrFromString("VeloxDestroyAfterRunFixture")
	wc := w32.WndClassExW{CbSize: uint32(unsafe.Sizeof(w32.WndClassExW{})), HInstance: instance,
		LpszClassName: class, LpfnWndProc: windows.NewCallback(wndproc)}
	w32.User32RegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, err := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, 0,
		0, 0, 0, 0, ^uintptr(2), 0, uintptr(instance), 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	browser := &bindingResponseBrowser{}
	thread, _, _ := w32.Kernel32GetCurrentThreadID.Call()
	view := &webview{hwnd: hwnd, mainthread: thread, browser: browser}
	setWindowContext(hwnd, view)
	t.Cleanup(func() {
		w32.User32DestroyWindow.Call(hwnd)
		deleteWindowContext(hwnd)
		// Destroying this private window can enqueue WM_QUIT after Run.
		for {
			r, _, _ := peek.Call(uintptr(unsafe.Pointer(&msg)), 0, w32.WMQuit, w32.WMQuit, 1)
			if r == 0 {
				break
			}
		}
	})
	view.Terminate()
	view.Run()
	view.Destroy()
	view.Destroy()
	valid, _, _ := user32.NewProc("IsWindow").Call(hwnd)
	if valid != 0 || getWindowContext(hwnd) != nil || browser.destroyed != 1 {
		t.Fatalf("Destroy after Run retained resources: window=%d context=%t browser destroy=%d",
			valid, getWindowContext(hwnd) != nil, browser.destroyed)
	}
	quit, _, _ := peek.Call(uintptr(unsafe.Pointer(&msg)), 0, w32.WMQuit, w32.WMQuit, 1)
	if quit != 0 {
		t.Fatal("Destroy after Run left a quit message for the next owner loop")
	}
}
