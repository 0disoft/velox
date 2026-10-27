//go:build windows

package edge

import (
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

// SDK 1.0.1722.45 WebView2.h: IUnknown followed by four get/put pairs.
var environmentOptionsIID = NewGUID("2FDE08A8-1E9A-4766-8C05-95A9CEB9D1C5")

const legacyReputationArgument = "--disable-features=msSmartScreenProtection"

type environmentOptionsVtbl struct {
	_IUnknownVtbl
	GetArguments, PutArguments         ComProc
	GetLanguage, PutLanguage           ComProc
	GetTargetVersion, PutTargetVersion ComProc
	GetSSO, PutSSO                     ComProc
}

// Pinned with Chromium's callback graph. Native AddRef/Release use the same
// owner registry, so options remain alive even if initialization is cancelled.
type environmentOptions struct {
	vtbl    *environmentOptionsVtbl
	impl    *Chromium
	mu      sync.Mutex
	strings [3]string
	sso     int32
}

var duplicateEnvironmentString = windows.NewLazySystemDLL("shlwapi.dll").NewProc("SHStrDupW")

func (o *environmentOptions) getString(index int, out **uint16) uintptr {
	if out == nil {
		return 0x80004003
	}
	*out = nil
	o.mu.Lock()
	value := o.strings[index]
	o.mu.Unlock()
	if value == "" {
		return 0
	}
	text, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return 0x80070057
	}
	// SHStrDupW returns a native CoTaskMem allocation owned by the caller.
	hr, _, _ := duplicateEnvironmentString.Call(uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(out)))
	return hr
}

func (o *environmentOptions) putString(index int, value *uint16) uintptr {
	text := windows.UTF16PtrToString(value)
	o.mu.Lock()
	o.strings[index] = text
	o.mu.Unlock()
	return 0
}

var environmentOptionsCallbacks = environmentOptionsVtbl{
	_IUnknownVtbl: _IUnknownVtbl{
		NewComProc(func(o *environmentOptions, iid *GUID, out *unsafe.Pointer) uintptr {
			return queryCallbackInterface(o.impl, unsafe.Pointer(o), iid, out, environmentOptionsIID)
		}),
		NewComProc(func(o *environmentOptions) uintptr { return o.impl.AddRef() }),
		NewComProc(func(o *environmentOptions) uintptr { return o.impl.Release() }),
	},
	GetArguments:     NewComProc(func(o *environmentOptions, out **uint16) uintptr { return o.getString(0, out) }),
	PutArguments:     NewComProc(func(o *environmentOptions, value *uint16) uintptr { return o.putString(0, value) }),
	GetLanguage:      NewComProc(func(o *environmentOptions, out **uint16) uintptr { return o.getString(1, out) }),
	PutLanguage:      NewComProc(func(o *environmentOptions, value *uint16) uintptr { return o.putString(1, value) }),
	GetTargetVersion: NewComProc(func(o *environmentOptions, out **uint16) uintptr { return o.getString(2, out) }),
	PutTargetVersion: NewComProc(func(o *environmentOptions, value *uint16) uintptr { return o.putString(2, value) }),
	GetSSO: NewComProc(func(o *environmentOptions, out *int32) uintptr {
		if out == nil {
			return 0x80004003
		}
		o.mu.Lock()
		*out = o.sso
		o.mu.Unlock()
		return 0
	}),
	PutSSO: NewComProc(func(o *environmentOptions, value int32) uintptr {
		o.mu.Lock()
		o.sso = 0
		if value != 0 {
			o.sso = 1
		}
		o.mu.Unlock()
		return 0
	}),
}

func needsLegacyReputationOption(version string) bool {
	fields := strings.Fields(version)
	if len(fields) == 0 {
		return false
	}
	parts := strings.Split(fields[0], ".")
	if len(parts) != 4 {
		return false
	}
	var major uint64
	for index, part := range parts {
		number, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return false
		}
		if index == 0 {
			major = number
		}
	}
	return major >= 92 && major < 112
}

func (e *Chromium) legacyReputationOptions() *environmentOptions {
	// The legacy flag is fixed at browser creation. Only the same explicit
	// local-content policy accepted by WebViewOptions may opt out.
	if !e.DisableReputationChecking || e.NavigationAllowed == nil || !e.DenyFrames || !e.DenyNewWindows || !e.DenyDownloads {
		return nil
	}
	version, err := webviewloader.GetAvailableVersion(e.BrowserExecutableFolder)
	if err != nil || !needsLegacyReputationOption(version) {
		return nil // Unknown/newer runtimes retain the Settings8/default path.
	}
	e.environmentOptions.strings[0] = legacyReputationArgument
	e.environmentOptions.strings[2] = strings.Fields(version)[0]
	return e.environmentOptions
}
