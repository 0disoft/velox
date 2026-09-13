//go:build windows

package edge

import (
	"errors"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ConfigureDevelopmentCache requests cache bypass on this development WebView.
// The in-process protocol call does not expose a debugging socket or clear storage.
func (e *Chromium) ConfigureDevelopmentCache(debug bool) error {
	if !debug {
		return nil
	}
	if e.destroyed || e.webview == nil {
		return errors.New("development cache policy requires an initialized WebView")
	}
	// No callback is retained: these fixed commands are dispatched asynchronously.
	// WebView2 accepts a null completion handler; native reload tests verify effect.
	for _, command := range [][2]string{
		{"Network.enable", `{}`},
		{"Network.setCacheDisabled", `{"cacheDisabled":true}`},
	} {
		method, _ := windows.UTF16PtrFromString(command[0])
		params, _ := windows.UTF16PtrFromString(command[1])
		result, _, _ := e.webview.vtbl.CallDevToolsProtocolMethod.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(unsafe.Pointer(method)),
			uintptr(unsafe.Pointer(params)), 0,
		)
		runtime.KeepAlive(method)
		runtime.KeepAlive(params)
		if err := hresult(result); err != nil {
			return err
		}
	}
	return nil
}
