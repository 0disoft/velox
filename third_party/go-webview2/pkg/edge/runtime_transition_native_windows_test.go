package edge

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

type blockedRuntimeObserver struct {
	*Chromium
	calls     int
	cancel    bool
	timerSeen bool
}

func (o *blockedRuntimeObserver) EnvironmentCompleted(hr uintptr, env *ICoreWebView2Environment) uintptr {
	o.calls++
	// Keep the real SDK environment and COM ownership, but model a requested
	// runtime that never replaces the old browser. Do not create a controller.
	o.runtimeTransition.required = "0.0.0.0"
	result := o.Chromium.EnvironmentCompleted(hr, env)
	o.timerSeen = o.timerSeen || o.runtimeTransition.timer != 0
	if o.calls == 1 {
		// Exercise the real timeout loop without spending the production 30s.
		o.runtimeTransition.deadline = time.Now().Add(250 * time.Millisecond)
		if o.cancel {
			o.runtimeTransition.deadline = time.Now().Add(5 * time.Second)
		}
	}
	if o.cancel && o.calls == 2 {
		o.Destroy()
		w32.User32PostMessageW.Call(o.hwnd, 0, 0, 0)
	}
	return result
}

func TestNativeRuntimeTransitionCleanup(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_RUNTIME_TRANSITION") != "1" {
		t.Skip("opt-in native transition timeout and cancellation")
	}
	for _, cancel := range []bool{false, true} {
		name := "timeout"
		if cancel {
			name = "cancel"
		}
		t.Run(name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			hr, _, _ := w32.Ole32CoInitializeEx.Call(0, 2)
			if err := hresult(hr); err != nil {
				t.Fatal(err)
			}
			defer windows.NewLazySystemDLL("ole32.dll").NewProc("CoUninitialize").Call()
			class := windows.StringToUTF16Ptr("STATIC")
			window, _, err := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, w32.WSOverlappedWindow, 0, 0, 320, 240, 0, 0, 0, 0)
			if window == 0 {
				t.Fatal(err)
			}
			defer w32.User32DestroyWindow.Call(window)
			e := NewChromium()
			defer e.Destroy()
			e.DataPath = filepath.Join(t.TempDir(), "profile")
			observer := &blockedRuntimeObserver{Chromium: e, cancel: cancel}
			e.envCompleted.impl = observer
			var watchdogFired atomic.Bool
			thread, _, _ := w32.Kernel32GetCurrentThreadID.Call()
			watchdogDone := make(chan struct{})
			watchdog := time.AfterFunc(8*time.Second, func() {
				defer close(watchdogDone)
				watchdogFired.Store(true)
				w32.User32PostThreadMessageW.Call(thread, w32.WMQuit, 0, 0)
			})
			initialized := e.Embed(window)
			if !watchdog.Stop() {
				<-watchdogDone
			}
			if initialized || watchdogFired.Load() || !observer.timerSeen || observer.calls < 1 {
				t.Fatalf("transition not exercised: ready=%t watchdog=%t timer=%t callbacks=%d", initialized, watchdogFired.Load(), observer.timerSeen, observer.calls)
			}
			if cancel && e.InitializationError() != nil || !cancel && !errors.Is(e.InitializationError(), windows.ERROR_TIMEOUT) {
				t.Fatalf("unexpected transition error: %v", e.InitializationError())
			}
			if !e.destroyed || e.runtimeTransition.timer != 0 || e.environment != nil || e.controller != nil || e.webview != nil {
				t.Fatal("transition retained timer or browser state")
			}
			if !pumpNativeCancellationUntil(10*time.Second, func() bool { return callbackReferenceCount(e) == 0 }) {
				t.Fatal("native transition callback references did not drain")
			}
			if !pumpNativeCancellationUntil(10*time.Second, func() bool { return os.RemoveAll(e.DataPath) == nil }) {
				t.Fatal("transition profile remains locked")
			}
			t.Logf("callbacks=%d timer=0 callback-refs=0 profile-released=true", observer.calls)
		})
	}
}
