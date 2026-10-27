package edge

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

const (
	runtimeTransitionTimeout = 30 * time.Second
	runtimeTransitionPoll    = 100 * time.Millisecond
	runtimeTransitionTimerID = 0x564c5254
)

var (
	runtimeSetTimer  = windows.NewLazySystemDLL("user32.dll").NewProc("SetTimer")
	runtimeKillTimer = windows.NewLazySystemDLL("user32.dll").NewProc("KillTimer")
)

// Owned only by the UI/COM thread. A rejected environment never receives a
// controller, so it cannot restart the old browser or write application data.
type runtimeTransition struct {
	required string
	deadline time.Time
	retryAt  time.Time
	timer    uintptr
}

func (t *runtimeTransition) expired(now time.Time) bool {
	return !t.deadline.IsZero() && !now.Before(t.deadline)
}

func (t *runtimeTransition) retryDue(now time.Time) bool {
	return !t.retryAt.IsZero() && !now.Before(t.retryAt)
}

func requestedRuntimeVersion(folder string, lookup func(string) (string, error)) string {
	version, err := lookup(folder)
	parts := strings.Fields(version)
	if err != nil || len(parts) == 0 {
		// Version discovery was optional before transition handling. Let the
		// SDK perform normal initialization (and report missing runtime errors).
		log.Printf("WebView2 runtime version unavailable; using SDK selection: %v", err)
		return ""
	}
	return parts[0]
}

func (e *Chromium) prepareRuntimeTransition() {
	version := requestedRuntimeVersion(e.BrowserExecutableFolder, webviewloader.GetAvailableVersion)
	if version != "" || e.runtimeTransition.deadline.IsZero() {
		e.runtimeTransition.required = version
	}
}

func (e *Chromium) acceptsRuntime(env *ICoreWebView2Environment) bool {
	if e.runtimeTransition.required == "" {
		return true
	}
	var version *uint16
	hr, _, _ := env.vtbl.GetBrowserVersionString.Call(uintptr(unsafe.Pointer(env)), uintptr(unsafe.Pointer(&version)))
	if version != nil {
		defer windows.CoTaskMemFree(unsafe.Pointer(version))
	}
	if err := hresult(hr); err != nil {
		e.failRuntimeTransition(fmt.Errorf("read active WebView2 runtime: %w", err))
		return false
	}
	actual := strings.Fields(windows.UTF16PtrToString(version))
	if len(actual) == 0 {
		e.failRuntimeTransition(fmt.Errorf("active WebView2 runtime returned no version"))
		return false
	}
	if actual[0] == e.runtimeTransition.required {
		return true
	}
	e.retryRuntimeEnvironment(actual[0])
	return false
}

func (e *Chromium) retryRuntimeEnvironment(actual string) {
	now := time.Now()
	if e.runtimeTransition.deadline.IsZero() {
		e.runtimeTransition.deadline = now.Add(runtimeTransitionTimeout)
		log.Printf("WebView2 waiting for runtime transition: active=%s requested=%s", actual, e.runtimeTransition.required)
		timer, _, err := runtimeSetTimer.Call(e.hwnd, runtimeTransitionTimerID, uintptr(runtimeTransitionPoll/time.Millisecond), 0)
		if timer == 0 {
			e.failRuntimeTransition(fmt.Errorf("start WebView2 transition timer: %w", err))
			return
		}
		e.runtimeTransition.timer = timer
	}
	e.runtimeTransition.retryAt = now.Add(runtimeTransitionPoll)
}

func (e *Chromium) failRuntimeTransition(err error) {
	e.initializationError = err
	atomic.StoreUintptr(&e.inited, 1)
	// A COM completion can run inside GetMessage. Wake that call explicitly.
	w32.User32PostMessageW.Call(e.hwnd, 0, 0, 0)
}

func (e *Chromium) stopRuntimeTransitionTimer() {
	if e.runtimeTransition.timer != 0 {
		runtimeKillTimer.Call(e.hwnd, e.runtimeTransition.timer)
		e.runtimeTransition.timer = 0
	}
}
