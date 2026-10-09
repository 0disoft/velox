//go:build windows

package edge

import (
	"fmt"
	"unsafe"
)

var processFailedIID = NewGUID("79E0AEA4-990B-42D9-AA1D-0FCC2E5BC7F1")

const browserProcessExitedKind = 0

type processFailedHandler struct {
	vtbl *processFailedVtbl
	impl *Chromium
}

type processFailedVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type processFailedArgs struct {
	vtbl *processFailedArgsVtbl
}

type processFailedArgsVtbl struct {
	_IUnknownVtbl
	GetProcessFailedKind ComProc
}

var processFailedCallbacks = processFailedVtbl{
	_IUnknownVtbl{
		NewComProc(func(this *processFailedHandler, iid *GUID, out *unsafe.Pointer) uintptr {
			return queryCallbackInterface(this.impl, unsafe.Pointer(this), iid, out, processFailedIID)
		}),
		NewComProc(func(this *processFailedHandler) uintptr { return this.impl.AddRef() }),
		NewComProc(func(this *processFailedHandler) uintptr { return this.impl.Release() }),
	},
	NewComProc(func(this *processFailedHandler, _ uintptr, args *processFailedArgs) uintptr {
		e := this.impl
		if e.destroyed || e.browserProcessExited || args == nil {
			return 0
		}
		var kind uint32
		result, _, _ := args.vtbl.GetProcessFailedKind.Call(uintptr(unsafe.Pointer(args)), uintptr(unsafe.Pointer(&kind)))
		// Never bypass document consent for an unreadable or non-browser failure.
		if hresult(result) == nil && kind == browserProcessExitedKind {
			e.browserProcessExited = true
			if e.BrowserProcessExitedCallback != nil {
				e.BrowserProcessExitedCallback()
			}
		}
		return 0
	}),
}

func (e *Chromium) registerProcessFailed() {
	if e.initializationError != nil || e.BrowserProcessExitedCallback == nil {
		return
	}
	result, _, _ := e.webview.vtbl.AddProcessFailed.Call(
		uintptr(unsafe.Pointer(e.webview)), uintptr(unsafe.Pointer(e.processFailed)),
		uintptr(unsafe.Pointer(&e.processFailedToken)))
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("register process failure: %w", err)
		return
	}
	e.processFailedRegistered = true
}
