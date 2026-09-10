//go:build windows

package edge

import (
	"fmt"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

func (e *Chromium) fileSystemAccessUsesBrowserConsent(args *iCoreWebView2PermissionRequestedEventArgs) bool {
	var userInitiated int32
	result, _, _ := args.vtbl.GetIsUserInitiated.Call(
		uintptr(unsafe.Pointer(args)), uintptr(unsafe.Pointer(&userInitiated)),
	)
	if err := hresult(result); err != nil {
		e.setPolicyError(fmt.Errorf("read permission user gesture: %w", err))
		return false
	}
	if userInitiated != 1 {
		return false
	}
	var uri *uint16
	result, _, _ = args.vtbl.GetURI.Call(
		uintptr(unsafe.Pointer(args)), uintptr(unsafe.Pointer(&uri)),
	)
	if err := hresult(result); err != nil {
		e.setPolicyError(fmt.Errorf("read permission origin: %w", err))
		return false
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(uri))
	return uri != nil && e.FileSystemAccessAllowed(w32.Utf16PtrToString(uri))
}
