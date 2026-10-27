package edge

import (
	"errors"
	"unsafe"
)

// SDK 1.0.1722.45 WebView2.h: Settings6 (the merged settings below),
// Settings7's PDF toolbar pair, then Settings8's reputation-checking pair.
type iCoreWebView2Settings8Vtbl struct {
	_ICoreWebViewSettingsVtbl
	GetHiddenPdfToolbarItems        ComProc
	PutHiddenPdfToolbarItems        ComProc
	GetIsReputationCheckingRequired ComProc
	PutIsReputationCheckingRequired ComProc
}

type iCoreWebView2Settings8 struct {
	vtbl *iCoreWebView2Settings8Vtbl
}

// SetReputationCheckingRequired applies to every WebView sharing the user-data
// folder on its next navigation. Unsupported runtimes retain their default
// protection and return supported=false. Other COM failures are not ignored.
func (i *ICoreWebViewSettings) SetReputationCheckingRequired(required bool) (supported bool, err error) {
	var settings *iCoreWebView2Settings8
	iid := NewGUID("{9E6B0E8F-86AD-4E81-8147-A9B5EDB68650}")
	hr, _, _ := i.vtbl.QueryInterface.Call(uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&settings)))
	if uint32(hr) == 0x80004002 { // E_NOINTERFACE: runtime predates Settings8.
		return false, nil
	}
	if err := hresult(hr); err != nil {
		return false, err
	}
	if settings == nil {
		return false, errors.New("WebView2 Settings8 query returned no interface")
	}
	defer settings.vtbl.Release.Call(uintptr(unsafe.Pointer(settings)))
	hr, _, _ = settings.vtbl.PutIsReputationCheckingRequired.Call(uintptr(unsafe.Pointer(settings)), uintptr(boolToInt(required)))
	return true, hresult(hr)
}
