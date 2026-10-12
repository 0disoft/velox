//go:build windows

package webview2

import (
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

func TestTerminateTargetsUIMessageLoop(t *testing.T) {
	for _, background := range []bool{false, true} {
		name := "ui-thread"
		if background {
			name = "background-thread"
		}
		t.Run(name, func(t *testing.T) {
			user32 := windows.NewLazySystemDLL("user32.dll")
			peek := user32.NewProc("PeekMessageW")
			ready := make(chan *webview, 1)
			started := make(chan struct{})
			stopped := make(chan struct{})
			release := make(chan struct{})
			go func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				var msg w32.Msg
				peek.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
				thread, _, _ := w32.Kernel32GetCurrentThreadID.Call()
				view := &webview{mainthread: thread}
				view.Dispatch(func() { close(started) })
				ready <- view
				view.Run()
				close(stopped)
				<-release
			}()
			view := <-ready
			t.Cleanup(func() {
				defer close(release)
				select {
				case <-stopped:
					return
				default:
				}
				// Recovery addresses only this fixture's thread, even when the
				// product sends WM_QUIT to the wrong queue.
				w32.User32PostThreadMessageW.Call(view.mainthread, w32.WMQuit, 0, 0)
				select {
				case <-stopped:
				case <-time.After(2 * time.Second):
					t.Error("fixture message loop did not stop during cleanup")
				}
			})
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("fixture message loop did not start")
			}
			if background {
				callerQuit := make(chan bool, 1)
				go func() {
					runtime.LockOSThread()
					defer runtime.UnlockOSThread()
					var msg w32.Msg
					peek.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
					view.Terminate()
					r, _, _ := peek.Call(uintptr(unsafe.Pointer(&msg)), 0, w32.WMQuit, w32.WMQuit, 1)
					callerQuit <- r != 0
				}()
				if <-callerQuit {
					t.Error("Terminate posted WM_QUIT to its background caller")
				}
			} else {
				view.Dispatch(view.Terminate)
			}
			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Error("Terminate did not stop the UI message loop")
			}
		})
	}
}
