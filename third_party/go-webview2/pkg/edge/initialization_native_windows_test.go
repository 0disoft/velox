//go:build windows

package edge

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

type nativeCancellationObserver struct {
	*Chromium
	environmentLate  bool
	environmentCalls int
	controllerLate   bool
	controllerCalls  int
	browser          windows.Handle
	failure          error
}

func (o *nativeCancellationObserver) EnvironmentCompleted(result uintptr, environment *ICoreWebView2Environment) uintptr {
	o.environmentCalls++
	o.environmentLate = o.destroyed
	if hresult(result) != nil || environment == nil {
		o.failure = fmt.Errorf("native environment completion failed: %08x", result)
	}
	return o.Chromium.EnvironmentCompleted(result, environment)
}

func (o *nativeCancellationObserver) CreateCoreWebView2ControllerCompleted(result uintptr, controller *ICoreWebView2Controller) uintptr {
	o.controllerCalls++
	o.controllerLate = o.destroyed
	if hresult(result) != nil || controller == nil {
		o.failure = fmt.Errorf("native controller completion failed: %08x", result)
	} else {
		var view *ICoreWebView2
		hr, _, _ := controller.vtbl.GetCoreWebView2.Call(uintptr(unsafe.Pointer(controller)), uintptr(unsafe.Pointer(&view)))
		if hresult(hr) != nil || view == nil {
			o.failure = fmt.Errorf("native controller returned no WebView: %08x", hr)
		} else {
			pid, err := view.GetBrowserProcessID()
			view.Release()
			if err != nil {
				o.failure = err
			} else {
				o.browser, err = windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
				if err != nil {
					o.failure = err
				}
			}
		}
	}
	return o.Chromium.CreateCoreWebView2ControllerCompleted(result, controller)
}

func TestNativeInitializationCancellation(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_CANCELLATION") != "1" {
		t.Skip("opt-in installed WebView2 cancellation test")
	}
	version, err := webviewloader.GetInstalledVersion()
	if err != nil || version == "" {
		t.Fatalf("installed WebView2 is required: %v", err)
	}
	t.Logf("source-fork evidence: WebView2=%s Go=%s arch=%s", version, runtime.Version(), runtime.GOARCH)
	for repetition := 1; repetition <= 3; repetition++ {
		for _, phase := range []string{"environment-completion", "controller-pending"} {
			t.Run(fmt.Sprintf("%s/%d", phase, repetition), func(t *testing.T) { runNativeCancellation(t, phase) })
		}
	}
}

func runNativeCancellation(t *testing.T, phase string) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := w32.Ole32CoInitializeEx.Call(0, 2)
	if err := hresult(hr); err != nil {
		t.Fatal(err)
	}
	defer windows.NewLazySystemDLL("ole32.dll").NewProc("CoUninitialize").Call()
	class, _ := windows.UTF16PtrFromString("STATIC")
	window, _, err := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, w32.WSOverlappedWindow, 0, 0, 320, 240, 0, 0, 0, 0)
	if window == 0 {
		t.Fatal(err)
	}
	defer w32.User32DestroyWindow.Call(window)
	profile := filepath.Join(t.TempDir(), "profile")
	e := NewChromium()
	e.DataPath = profile
	observer := &nativeCancellationObserver{Chromium: e}
	e.envCompleted.impl = observer
	e.controllerCompleted.impl = observer
	defer func() {
		e.Destroy()
		if observer.browser != 0 {
			_ = windows.CloseHandle(observer.browser)
		}
	}()
	requested := false
	e.StartupPhase = func(name string) {
		if name == "environment-created" {
			requested = true
			if phase == "environment-completion" {
				e.Destroy()
			}
			w32.User32PostQuitMessage.Call(0)
		}
	}

	var watchdogFired atomic.Bool
	// Keep a missing runtime callback from leaving Embed blocked in GetMessage.
	threadID, _, _ := w32.Kernel32GetCurrentThreadID.Call()
	stopWatchdog := make(chan struct{})
	watchdogDone := make(chan struct{})
	go func() {
		defer close(watchdogDone)
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		select {
		case <-stopWatchdog:
			return
		case <-timer.C:
			watchdogFired.Store(true)
			w32.User32PostThreadMessageW.Call(threadID, w32.WMQuit, 0, 0)
		}
	}()
	initialized := e.Embed(window)
	close(stopWatchdog)
	<-watchdogDone
	e.Destroy()
	runtime.GC()

	// Deliver native completions after Destroy on the original STA, until every
	// runtime-held callback reference is released. No synthetic Invoke is used.
	drained := pumpNativeCancellationUntil(10*time.Second, func() bool { return callbackReferenceCount(e) == 0 })
	if !drained {
		t.Fatalf("native callback references did not drain: %d", callbackReferenceCount(e))
	}
	if watchdogFired.Load() || !requested || initialized {
		t.Fatalf("cancellation failed: requested=%t initialized=%t watchdog=%t", requested, initialized, watchdogFired.Load())
	}
	if observer.failure != nil {
		t.Fatal(observer.failure)
	}
	if observer.environmentCalls != 1 {
		t.Fatalf("environment callbacks=%d", observer.environmentCalls)
	}
	if phase == "environment-completion" {
		if observer.environmentLate || observer.controllerCalls != 0 {
			t.Fatal("environment cancellation scheduled a controller or missed its phase")
		}
	} else if observer.environmentLate || observer.controllerCalls != 1 || !observer.controllerLate || observer.browser == 0 {
		t.Fatal("controller completion did not arrive after cancellation")
	}
	if !e.destroyed || e.environment != nil || e.controller != nil || e.webview != nil || e.inited != 0 {
		t.Fatal("late callback revived destroyed browser")
	}
	if observer.browser != 0 && !pumpNativeCancellationUntil(10*time.Second, func() bool {
		state, err := windows.WaitForSingleObject(observer.browser, 0)
		return err == nil && state == windows.WAIT_OBJECT_0
	}) {
		t.Fatal("late controller browser did not exit")
	}
	if !pumpNativeCancellationUntil(10*time.Second, func() bool { return os.RemoveAll(profile) == nil }) {
		t.Fatal("canceled profile remains locked")
	}
	runtime.GC()
	if callbackReferenceCount(e) != 0 {
		t.Fatal("callback owner was retained after cleanup")
	}
	t.Logf("native cancellation=%s late-controller=%t callback-refs=0 profile-released=true browser-exit-observed=%t", phase, observer.controllerLate, observer.browser != 0)
}

func pumpNativeCancellationUntil(timeout time.Duration, done func() bool) bool {
	peek := windows.NewLazySystemDLL("user32.dll").NewProc("PeekMessageW")
	deadline := time.Now().Add(timeout)
	for {
		var message w32.Msg
		for count := 0; count < 100; count++ {
			found, _, _ := peek.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, 1)
			if found == 0 {
				break
			}
			if message.Message != w32.WMQuit {
				w32.User32TranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
				w32.User32DispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
			}
		}
		if done() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}
