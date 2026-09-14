//go:build windows

package edge

import (
	"fmt"
	"unsafe"
)

var windowCloseRequestedIID = NewGUID("5C19E9E0-092F-486B-AFFA-CA8231913039")

type windowCloseRequestedHandler struct {
	vtbl *windowCloseRequestedVtbl
	impl *Chromium
}

type windowCloseRequestedVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

var windowCloseRequestedCallbacks = windowCloseRequestedVtbl{
	_IUnknownVtbl{
		NewComProc(func(this *windowCloseRequestedHandler, iid *GUID, out *unsafe.Pointer) uintptr {
			return queryCallbackInterface(this.impl, unsafe.Pointer(this), iid, out, windowCloseRequestedIID)
		}),
		NewComProc(func(this *windowCloseRequestedHandler) uintptr { return this.impl.AddRef() }),
		NewComProc(func(this *windowCloseRequestedHandler) uintptr { return this.impl.Release() }),
	},
	NewComProc(func(this *windowCloseRequestedHandler, _ uintptr, _ uintptr) uintptr {
		if !this.impl.destroyed && this.impl.WindowCloseRequestedCallback != nil {
			this.impl.WindowCloseRequestedCallback()
		}
		return 0
	}),
}

func (e *Chromium) registerWindowCloseRequested() {
	if e.initializationError != nil || e.WindowCloseRequestedCallback == nil {
		return
	}
	result, _, _ := e.webview.vtbl.AddWindowCloseRequested.Call(
		uintptr(unsafe.Pointer(e.webview)), uintptr(unsafe.Pointer(e.windowCloseRequested)),
		uintptr(unsafe.Pointer(&e.windowCloseToken)))
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("register window close request: %w", err)
		return
	}
	e.windowCloseRegistered = true
}
