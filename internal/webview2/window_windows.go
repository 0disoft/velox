//go:build windows

package webview2

import (
	"errors"

	webview "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	showMinimized = 6
	showMaximized = 3
	showRestored  = 9
)

var (
	user32Window = windows.NewLazySystemDLL("user32.dll")
	showWindow   = user32Window.NewProc("ShowWindow")
	isIconic     = user32Window.NewProc("IsIconic")
	isZoomed     = user32Window.NewProc("IsZoomed")
)

type nativeWindow struct {
	view    webview.WebView
	runtime *Runtime
}

func (w nativeWindow) State() (string, error) {
	handle, err := w.handle()
	if err != nil {
		return "", err
	}
	if result, _, _ := isIconic.Call(handle); result != 0 {
		return "minimized", nil
	}
	if result, _, _ := isZoomed.Call(handle); result != 0 {
		return "maximized", nil
	}
	return "normal", nil
}

func (w nativeWindow) Minimize() error { return w.show(showMinimized) }
func (w nativeWindow) Maximize() error { return w.show(showMaximized) }
func (w nativeWindow) Restore() error  { return w.show(showRestored) }

func (w nativeWindow) Close() error {
	if _, err := w.handle(); err != nil {
		return err
	}
	// The first dispatch is queued before the binding response. Calling Close
	// from it places the shutdown sequence behind that response.
	w.view.Dispatch(w.runtime.Close)
	return nil
}

func (w nativeWindow) show(command uintptr) error {
	handle, err := w.handle()
	if err != nil {
		return err
	}
	showWindow.Call(handle, command)
	return nil
}

func (w nativeWindow) handle() (uintptr, error) {
	if w.view == nil || w.runtime == nil {
		return 0, errors.New("native window is unavailable")
	}
	handle := uintptr(w.view.Window())
	if handle == 0 {
		return 0, errors.New("native window handle is unavailable")
	}
	return handle, nil
}
