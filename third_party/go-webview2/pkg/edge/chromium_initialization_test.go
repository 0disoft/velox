//go:build windows

package edge

import (
	"errors"
	"sync/atomic"
	"testing"
)

func TestFailedInitializationDoesNotAccessWebView(t *testing.T) {
	for _, tc := range []struct {
		name     string
		complete uintptr
		err      error
	}{
		{"message-loop-stopped", 0, nil},
		{"missing-webview", 1, nil},
		{"failed-completion", 1, errors.New("injected initialization failure")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &Chromium{inited: tc.complete, initializationError: tc.err}
			if e.finishInitialization() || !e.destroyed {
				t.Fatal("incomplete initialization must fail and tear down")
			}
			e.Destroy()
		})
	}
}

func TestCompletionRejectsFailedHRESULTAndEmptyInterface(t *testing.T) {
	for _, result := range []uintptr{0x80004005, 0} {
		var env *ICoreWebView2Environment
		var controller *ICoreWebView2Controller
		if result != 0 {
			env = &ICoreWebView2Environment{}
			controller = &ICoreWebView2Controller{}
		}
		e := &Chromium{}
		e.EnvironmentCompleted(result, env)
		if e.initializationError == nil || atomic.LoadUintptr(&e.inited) != 1 {
			t.Fatalf("environment completion %x was not rejected", result)
		}
		e = &Chromium{}
		e.CreateCoreWebView2ControllerCompleted(result, controller)
		if e.initializationError == nil || atomic.LoadUintptr(&e.inited) != 1 {
			t.Fatalf("controller completion %x was not rejected", result)
		}
	}
}

func TestLateControllerIsClosedWithoutTakingOwnership(t *testing.T) {
	closed := 0
	controller := &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		Close: NewComProc(func(this *ICoreWebView2Controller) uintptr {
			closed++
			return 0
		}),
	}}
	e := &Chromium{}
	e.Destroy()
	e.CreateCoreWebView2ControllerCompleted(0, controller)
	if closed != 1 || e.controller != nil {
		t.Fatal("late borrowed controller must be closed without retaining it")
	}
}

func TestLateInitializationCannotReviveDestroyedBrowser(t *testing.T) {
	e := &Chromium{}
	e.Destroy()
	e.EnvironmentCompleted(0, nil)
	e.CreateCoreWebView2ControllerCompleted(0, nil)
	if e.environment != nil || e.controller != nil || e.webview != nil || e.inited != 0 {
		t.Fatal("late completion revived destroyed browser")
	}
	if e.Embed(0) || e.finishInitialization() {
		t.Fatal("destroyed browser must not restart")
	}
}
