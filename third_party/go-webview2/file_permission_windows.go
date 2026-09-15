//go:build windows

package webview2

import (
	"errors"
	"github.com/jchv/go-webview2/pkg/edge"
)

// FilePermission is a native-only maintenance operation. It must run on the UI
// thread; reset requires explicit native user consent. No JavaScript binding.
func FilePermission(view WebView, origin string, reset bool, done func(uint32, error)) error {
	w, ok := view.(*webview)
	if !ok || done == nil {
		return errors.New("file permission management is unavailable")
	}
	browser, ok := w.browser.(*edge.Chromium)
	if !ok {
		return errors.New("file permission management is unavailable")
	}
	return browser.FilePermission(origin, reset, func(state edge.CoreWebView2PermissionState, err error) { done(uint32(state), err) })
}
