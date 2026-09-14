//go:build windows

package edge

import (
	"testing"
	"unsafe"
)

func TestWindowCloseCallbackIgnoresLateDelivery(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	e.AddRef()
	defer e.Release()
	calls := 0
	e.WindowCloseRequestedCallback = func() { calls++ }
	invoke := func() {
		result, _, _ := e.windowCloseRequested.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.windowCloseRequested)), 0, 0)
		if result != 0 {
			t.Fatalf("close callback HRESULT=%x", result)
		}
	}
	invoke()
	e.Destroy()
	invoke()
	if calls != 1 || callbackReferenceCount(e) != 1 {
		t.Fatalf("calls=%d refs=%d", calls, callbackReferenceCount(e))
	}
}
