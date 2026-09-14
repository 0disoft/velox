//go:build windows

package edge

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestControllerCloseUsesHRESULT(t *testing.T) {
	setLastError := windows.NewLazySystemDLL("kernel32.dll").NewProc("SetLastError")
	for _, tc := range []struct {
		name      string
		result    uintptr
		lastError uintptr
	}{
		{"success", 0, 0},
		{"success-with-stale-last-error", 0, 5},
		{"nonzero-success", 1, 0},
		{"failed-hresult", 0x80004005, 0},
		{"access-denied-hresult", 0x80070005, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			controller := &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
				Close: NewComProc(func(uintptr) uintptr {
					calls++
					setLastError.Call(tc.lastError)
					return tc.result
				}),
			}}
			got, want := controller.Close(), hresult(tc.result)
			if calls != 1 || (got == nil) != (want == nil) || (got != nil && got.Error() != want.Error()) {
				t.Fatalf("Close: calls=%d error=%v want=%v", calls, got, want)
			}
		})
	}
}

func TestDestroyContinuesAfterControllerCloseFailure(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	defer e.Destroy()
	var calls, phases []string
	release := func(name string) ComProc {
		return NewComProc(func(uintptr) uintptr { calls = append(calls, name); return 0 })
	}
	e.controller = &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		_IUnknownVtbl: _IUnknownVtbl{Release: release("controller")},
		Close:         NewComProc(func(uintptr) uintptr { calls = append(calls, "close"); return 0x80004005 }),
	}}
	e.webview = &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("webview")}}}
	e.environment = &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("environment")}}}
	e.ShutdownPhase = func(name string) { phases = append(phases, name) }
	e.Destroy()
	e.Destroy()
	if strings.Join(calls, ",") != "close,webview,controller,environment" {
		t.Fatalf("teardown order/calls = %v", calls)
	}
	if strings.Join(phases, ",") != "chromium-destroy-entered,event-handlers-removed,controller-close-failed,webview-released,controller-released,environment-released" {
		t.Fatalf("shutdown phases = %v", phases)
	}
	if e.controller != nil || e.webview != nil || e.environment != nil || callbackReferenceCount(e) != 0 {
		t.Fatal("failed close skipped interface or callback owner release")
	}
}

func TestLateControllerCloseFailureIsReportedWithoutOwnership(t *testing.T) {
	e := NewChromium()
	e.Destroy()
	var phases []string
	e.ShutdownPhase = func(name string) { phases = append(phases, name) }
	closed := 0
	controller := &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		Close: NewComProc(func(uintptr) uintptr { closed++; return 0x80004005 }),
	}}
	e.CreateCoreWebView2ControllerCompleted(0, controller)
	if closed != 1 || strings.Join(phases, ",") != "controller-close-failed" || e.controller != nil || e.webview != nil || e.environment != nil || e.inited != 0 {
		t.Fatalf("late failed close: calls=%d phases=%v", closed, phases)
	}
}
