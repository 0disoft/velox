package edge

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

// Sent messages run inside GetMessage, which then continues waiting for a queued
// message. Model COM's reentrant completion without relying on SDK timing.
func TestInitializationFailureWakesBlockedMessagePump(t *testing.T) {
	for _, phase := range []string{"environment", "controller"} {
		t.Run(phase, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			e := NewChromium()
			defer e.Destroy()
			user32 := windows.NewLazySystemDLL("user32.dll")
			const completionMessage = w32.WMApp + 72
			callback := windows.NewCallback(func(hwnd uintptr, message uint32, wparam, lparam uintptr) uintptr {
				if message == completionMessage {
					if phase == "environment" {
						e.EnvironmentCompleted(0x80004005, nil)
					} else {
						e.CreateCoreWebView2ControllerCompleted(0x80004005, nil)
					}
					return 0
				}
				result, _, _ := w32.User32DefWindowProcW.Call(hwnd, uintptr(message), wparam, lparam)
				return result
			})
			var instance windows.Handle
			if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
				t.Fatal(err)
			}
			class := windows.StringToUTF16Ptr(fmt.Sprintf("VeloxInitWake-%s-%d", phase, time.Now().UnixNano()))
			wc := w32.WndClassExW{CbSize: uint32(unsafe.Sizeof(w32.WndClassExW{})), HInstance: instance, LpszClassName: class, LpfnWndProc: callback}
			if result, _, err := w32.User32RegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); result == 0 {
				t.Fatal(err)
			}
			defer user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(class)), uintptr(instance))
			window, _, err := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, uintptr(instance), 0)
			if window == 0 {
				t.Fatal(err)
			}
			defer w32.User32DestroyWindow.Call(window)
			e.hwnd = window
			var message w32.Msg
			// This private message-only window has no paint/input producers.
			peek := user32.NewProc("PeekMessageW")
			for {
				result, _, _ := peek.Call(uintptr(unsafe.Pointer(&message)), window, 0, 0, 1)
				if result == 0 {
					break
				}
			}
			senderDone := make(chan struct{})
			go func() {
				defer close(senderDone)
				var result uintptr
				user32.NewProc("SendMessageTimeoutW").Call(window, completionMessage, 0, 0, 3, 5000, uintptr(unsafe.Pointer(&result)))
			}()
			var rescued atomic.Bool
			rescueDone := make(chan struct{})
			rescue := time.AfterFunc(2*time.Second, func() {
				defer close(rescueDone)
				rescued.Store(true)
				w32.User32PostMessageW.Call(window, 0, 0, 0)
			})
			result, _, waitErr := w32.User32GetMessageW.Call(uintptr(unsafe.Pointer(&message)), window, 0, 0)
			if !rescue.Stop() {
				<-rescueDone
			}
			<-senderDone
			if result == 0 || result == ^uintptr(0) || atomic.LoadUintptr(&e.inited) != 1 || e.InitializationError() == nil {
				t.Fatalf("failure completion was not delivered: result=%x error=%v init=%v", result, waitErr, e.InitializationError())
			}
			if rescued.Load() {
				t.Fatal("initialization failed but GetMessage stayed blocked until watchdog rescue")
			}
			if e.finishInitialization() || !e.destroyed || callbackReferenceCount(e) != 0 {
				t.Fatal("failed initialization did not release its owner")
			}
		})
	}
}
