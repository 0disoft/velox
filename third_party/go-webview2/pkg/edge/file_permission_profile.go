//go:build windows

package edge

import (
	"errors"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type filePermissionProfileVtbl struct {
	_IUnknownVtbl
	profileMethods                  [7]ComProc
	profile2Methods                 [3]ComProc
	profile3Methods                 [2]ComProc
	SetPermissionState              ComProc
	GetNonDefaultPermissionSettings ComProc
}

type filePermissionProfile struct{ vtbl *filePermissionProfileVtbl }
type profileGetter struct {
	vtbl *struct {
		inherited  [105]ComProc
		GetProfile ComProc
	}
}
type permissionCollection struct {
	vtbl *struct {
		_IUnknownVtbl
		GetValueAtIndex, GetCount ComProc
	}
}
type permissionSetting struct {
	vtbl *struct {
		_IUnknownVtbl
		GetKind, GetOrigin, GetState ComProc
	}
}

func getFilePermissionProfile(view *ICoreWebView2) (*filePermissionProfile, error) {
	var getter *profileGetter
	iid := NewGUID("F75F09A8-667E-4983-88D6-C8773F315E84")
	hr, _, _ := view.vtbl.QueryInterface.Call(uintptr(unsafe.Pointer(view)), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&getter)))
	if err := hresult(hr); err != nil {
		return nil, err
	}
	if getter == nil {
		return nil, errors.New("WebView2 profile interface is unavailable")
	}
	defer releaseIUnknown(uintptr(unsafe.Pointer(getter)))
	var profile *iUnknown
	hr, _, _ = getter.vtbl.GetProfile.Call(uintptr(unsafe.Pointer(getter)), uintptr(unsafe.Pointer(&profile)))
	if err := hresult(hr); err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, errors.New("WebView2 profile is unavailable")
	}
	defer releaseIUnknown(uintptr(unsafe.Pointer(profile)))
	var result *filePermissionProfile
	iid = NewGUID("8F4AE680-192E-4EC8-833A-21CFADAEF628")
	hr, _, _ = profile.vtbl.QueryInterface.Call(uintptr(unsafe.Pointer(profile)), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&result)))
	runtime.KeepAlive(iid)
	if err := hresult(hr); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("WebView2 permission management is unavailable")
	}
	return result, nil
}

func (p *filePermissionProfile) read(handler *filePermissionHandler) error {
	hr, _, _ := p.vtbl.GetNonDefaultPermissionSettings.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(handler)))
	return hresult(hr)
}

func (p *filePermissionProfile) reset(origin string, handler *filePermissionHandler) error {
	uri, err := windows.UTF16PtrFromString(origin)
	if err != nil {
		return err
	}
	hr, _, _ := p.vtbl.SetPermissionState.Call(uintptr(unsafe.Pointer(p)), uintptr(CoreWebView2PermissionKindFileReadWrite), uintptr(unsafe.Pointer(uri)), uintptr(CoreWebView2PermissionStateDefault), uintptr(unsafe.Pointer(handler)))
	runtime.KeepAlive(uri)
	return hresult(hr)
}

func readFilePermission(collection *permissionCollection, origin string) (CoreWebView2PermissionState, error) {
	if collection == nil {
		return 0, errors.New("permission snapshot is missing")
	}
	var count uint32
	hr, _, _ := collection.vtbl.GetCount.Call(uintptr(unsafe.Pointer(collection)), uintptr(unsafe.Pointer(&count)))
	if err := hresult(hr); err != nil {
		return 0, err
	}
	if count > 4096 {
		return 0, errors.New("permission snapshot exceeds limit")
	}
	state := CoreWebView2PermissionStateDefault
	matched := false
	for i := uint32(0); i < count; i++ {
		var item *permissionSetting
		hr, _, _ = collection.vtbl.GetValueAtIndex.Call(uintptr(unsafe.Pointer(collection)), uintptr(i), uintptr(unsafe.Pointer(&item)))
		if err := hresult(hr); err != nil {
			return 0, err
		}
		if item == nil {
			return 0, errors.New("permission setting is missing")
		}
		kind, uri, value, err := readPermissionSetting(item)
		releaseIUnknown(uintptr(unsafe.Pointer(item)))
		if err != nil {
			return 0, err
		}
		if kind == CoreWebView2PermissionKindFileReadWrite && strings.TrimSuffix(uri, "/") == origin {
			if matched {
				return 0, errors.New("duplicate file permission setting")
			}
			matched, state = true, value
		}
	}
	return state, nil
}

func readPermissionSetting(item *permissionSetting) (CoreWebView2PermissionKind, string, CoreWebView2PermissionState, error) {
	var kind CoreWebView2PermissionKind
	var state CoreWebView2PermissionState
	var uri *uint16
	p := uintptr(unsafe.Pointer(item))
	hr, _, _ := item.vtbl.GetKind.Call(p, uintptr(unsafe.Pointer(&kind)))
	if err := hresult(hr); err != nil {
		return 0, "", 0, err
	}
	hr, _, _ = item.vtbl.GetOrigin.Call(p, uintptr(unsafe.Pointer(&uri)))
	if uri != nil {
		defer windows.CoTaskMemFree(unsafe.Pointer(uri))
	}
	if err := hresult(hr); err != nil {
		return 0, "", 0, err
	}
	if uri == nil {
		return 0, "", 0, errors.New("permission origin is missing")
	}
	hr, _, _ = item.vtbl.GetState.Call(p, uintptr(unsafe.Pointer(&state)))
	if err := hresult(hr); err != nil {
		return 0, "", 0, err
	}
	if state > CoreWebView2PermissionStateDeny {
		return 0, "", 0, errors.New("invalid permission state")
	}
	return kind, windows.UTF16PtrToString(uri), state, nil
}
