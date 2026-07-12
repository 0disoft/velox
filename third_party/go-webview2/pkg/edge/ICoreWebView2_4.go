//go:build windows

package edge

import "unsafe"

type iCoreWebView2_4Vtbl struct {
	iCoreWebView2_3Vtbl
	AddFrameCreated        ComProc
	RemoveFrameCreated     ComProc
	AddDownloadStarting    ComProc
	RemoveDownloadStarting ComProc
}

type ICoreWebView2_4 struct {
	vtbl *iCoreWebView2_4Vtbl
}

func (i *ICoreWebView2_4) AddDownloadStarting(handler *downloadStartingEventHandler, token *_EventRegistrationToken) error {
	result, _, _ := i.vtbl.AddDownloadStarting.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(handler)),
		uintptr(unsafe.Pointer(token)),
	)
	return hresult(result)
}

func (i *ICoreWebView2_4) RemoveDownloadStarting(token _EventRegistrationToken) error {
	result, _, _ := i.vtbl.RemoveDownloadStarting.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(token.Value),
	)
	return hresult(result)
}

func (i *ICoreWebView2_4) Release() uintptr {
	result, _, _ := i.vtbl.Release.Call(uintptr(unsafe.Pointer(i)))
	return result
}

func (i *ICoreWebView2) GetICoreWebView2_4() *ICoreWebView2_4 {
	var result *ICoreWebView2_4
	iid := NewGUID("{20D02D59-6DF2-42DC-BD06-F98A694B1302}")
	hresultValue, _, _ := i.vtbl.QueryInterface.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&result)),
	)
	if hresult(hresultValue) != nil {
		return nil
	}
	return result
}
