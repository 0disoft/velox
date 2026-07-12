//go:build windows

package edge

import (
	"fmt"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

type iCoreWebView2NavigationStartingEventArgsVtbl struct {
	_IUnknownVtbl
	GetURI             ComProc
	GetIsUserInitiated ComProc
	GetIsRedirected    ComProc
	GetRequestHeaders  ComProc
	GetCancel          ComProc
	PutCancel          ComProc
	GetNavigationID    ComProc
}

type iCoreWebView2NavigationStartingEventArgs struct {
	vtbl *iCoreWebView2NavigationStartingEventArgsVtbl
}

func (i *iCoreWebView2NavigationStartingEventArgs) URI() (string, error) {
	var value *uint16
	result, _, _ := i.vtbl.GetURI.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(&value)),
	)
	if err := hresult(result); err != nil {
		return "", err
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(value))
	return w32.Utf16PtrToString(value), nil
}

func (i *iCoreWebView2NavigationStartingEventArgs) PutCancel(cancel bool) error {
	result, _, _ := i.vtbl.PutCancel.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(boolToInt(cancel)),
	)
	return hresult(result)
}

type navigationStartingEventHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type navigationStartingEventHandler struct {
	vtbl  *navigationStartingEventHandlerVtbl
	owner *Chromium
	frame bool
}

func navigationStartingQueryInterface(this *navigationStartingEventHandler, _, _ uintptr) uintptr {
	return this.owner.QueryInterface(0, 0)
}

func navigationStartingAddRef(this *navigationStartingEventHandler) uintptr {
	return this.owner.AddRef()
}

func navigationStartingRelease(this *navigationStartingEventHandler) uintptr {
	return this.owner.Release()
}

func navigationStartingInvoke(this *navigationStartingEventHandler, _ *ICoreWebView2, args *iCoreWebView2NavigationStartingEventArgs) uintptr {
	this.owner.handleNavigationStarting(args, this.frame)
	return 0
}

var navigationStartingEventHandlerVTable = navigationStartingEventHandlerVtbl{
	_IUnknownVtbl: _IUnknownVtbl{
		QueryInterface: NewComProc(navigationStartingQueryInterface),
		AddRef:         NewComProc(navigationStartingAddRef),
		Release:        NewComProc(navigationStartingRelease),
	},
	Invoke: NewComProc(navigationStartingInvoke),
}

func newNavigationStartingEventHandler(owner *Chromium, frame bool) *navigationStartingEventHandler {
	return &navigationStartingEventHandler{
		vtbl:  &navigationStartingEventHandlerVTable,
		owner: owner,
		frame: frame,
	}
}

type iCoreWebView2NewWindowRequestedEventArgsVtbl struct {
	_IUnknownVtbl
	GetURI             ComProc
	PutNewWindow       ComProc
	GetNewWindow       ComProc
	PutHandled         ComProc
	GetHandled         ComProc
	GetIsUserInitiated ComProc
	GetDeferral        ComProc
}

type iCoreWebView2NewWindowRequestedEventArgs struct {
	vtbl *iCoreWebView2NewWindowRequestedEventArgsVtbl
}

func (i *iCoreWebView2NewWindowRequestedEventArgs) PutHandled(handled bool) error {
	result, _, _ := i.vtbl.PutHandled.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(boolToInt(handled)),
	)
	return hresult(result)
}

type newWindowRequestedEventHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type newWindowRequestedEventHandler struct {
	vtbl  *newWindowRequestedEventHandlerVtbl
	owner *Chromium
}

func newWindowRequestedQueryInterface(this *newWindowRequestedEventHandler, _, _ uintptr) uintptr {
	return this.owner.QueryInterface(0, 0)
}

func newWindowRequestedAddRef(this *newWindowRequestedEventHandler) uintptr {
	return this.owner.AddRef()
}

func newWindowRequestedRelease(this *newWindowRequestedEventHandler) uintptr {
	return this.owner.Release()
}

func newWindowRequestedInvoke(this *newWindowRequestedEventHandler, _ *ICoreWebView2, args *iCoreWebView2NewWindowRequestedEventArgs) uintptr {
	if err := args.PutHandled(true); err != nil {
		this.owner.setPolicyError(fmt.Errorf("deny new window: %w", err))
	} else {
		this.owner.reportPolicyBlocked("new-window")
	}
	return 0
}

var newWindowRequestedEventHandlerVTable = newWindowRequestedEventHandlerVtbl{
	_IUnknownVtbl: _IUnknownVtbl{
		QueryInterface: NewComProc(newWindowRequestedQueryInterface),
		AddRef:         NewComProc(newWindowRequestedAddRef),
		Release:        NewComProc(newWindowRequestedRelease),
	},
	Invoke: NewComProc(newWindowRequestedInvoke),
}

func newNewWindowRequestedEventHandler(owner *Chromium) *newWindowRequestedEventHandler {
	return &newWindowRequestedEventHandler{vtbl: &newWindowRequestedEventHandlerVTable, owner: owner}
}

type iCoreWebView2DownloadStartingEventArgsVtbl struct {
	_IUnknownVtbl
	GetDownloadOperation ComProc
	GetCancel            ComProc
	PutCancel            ComProc
	GetResultFilePath    ComProc
	PutResultFilePath    ComProc
	GetHandled           ComProc
	PutHandled           ComProc
	GetDeferral          ComProc
}

type iCoreWebView2DownloadStartingEventArgs struct {
	vtbl *iCoreWebView2DownloadStartingEventArgsVtbl
}

func (i *iCoreWebView2DownloadStartingEventArgs) PutCancel(cancel bool) error {
	result, _, _ := i.vtbl.PutCancel.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(boolToInt(cancel)),
	)
	return hresult(result)
}

func (i *iCoreWebView2DownloadStartingEventArgs) PutHandled(handled bool) error {
	result, _, _ := i.vtbl.PutHandled.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(boolToInt(handled)),
	)
	return hresult(result)
}

type downloadStartingEventHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type downloadStartingEventHandler struct {
	vtbl  *downloadStartingEventHandlerVtbl
	owner *Chromium
}

func downloadStartingQueryInterface(this *downloadStartingEventHandler, _, _ uintptr) uintptr {
	return this.owner.QueryInterface(0, 0)
}

func downloadStartingAddRef(this *downloadStartingEventHandler) uintptr {
	return this.owner.AddRef()
}

func downloadStartingRelease(this *downloadStartingEventHandler) uintptr {
	return this.owner.Release()
}

func downloadStartingInvoke(this *downloadStartingEventHandler, _ *ICoreWebView2, args *iCoreWebView2DownloadStartingEventArgs) uintptr {
	if err := args.PutCancel(true); err != nil {
		this.owner.setPolicyError(fmt.Errorf("cancel download: %w", err))
	}
	if err := args.PutHandled(true); err != nil {
		this.owner.setPolicyError(fmt.Errorf("hide download UI: %w", err))
	}
	this.owner.reportPolicyBlocked("download")
	return 0
}

var downloadStartingEventHandlerVTable = downloadStartingEventHandlerVtbl{
	_IUnknownVtbl: _IUnknownVtbl{
		QueryInterface: NewComProc(downloadStartingQueryInterface),
		AddRef:         NewComProc(downloadStartingAddRef),
		Release:        NewComProc(downloadStartingRelease),
	},
	Invoke: NewComProc(downloadStartingInvoke),
}

func newDownloadStartingEventHandler(owner *Chromium) *downloadStartingEventHandler {
	return &downloadStartingEventHandler{vtbl: &downloadStartingEventHandlerVTable, owner: owner}
}

func hresult(result uintptr) error {
	if int32(result) >= 0 {
		return nil
	}
	return fmt.Errorf("HRESULT 0x%08X", uint32(result))
}
