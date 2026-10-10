//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"unsafe"

	"github.com/0disoft/velox/internal/webview2"
	"golang.org/x/sys/windows"
)

type startupFailureKind uint8

const (
	startupHost startupFailureKind = iota
	startupConfiguration
	startupProfile
	startupRuntime
	startupDisplay
	startupInstance
)

type startupFailureReporter struct {
	output   io.Writer
	notice   func(startupFailureKind)
	headless bool
}

func (r startupFailureReporter) notify(kind startupFailureKind) {
	if !r.headless && r.notice != nil {
		r.notice(kind)
	}
}

func (r startupFailureReporter) fail(kind startupFailureKind, err error, code int) int {
	fmt.Fprintf(r.output, "velox-host: %v\n", err)
	r.notify(kind)
	return code
}

func (r startupFailureReporter) runtimeFailure(err error) int {
	if errors.Is(err, webview2.ErrInitializationCanceled) {
		return 0
	}
	code := 6
	if errors.Is(err, webview2.ErrRuntimeUnavailable) {
		code = 5
	}
	return r.fail(startupRuntime, err, code)
}

func startupFailureMessage(kind startupFailureKind) string {
	switch kind {
	case startupConfiguration:
		return "The application configuration or packaged assets could not be loaded.\n\nExtract the complete app package into a folder and try again. If it still fails, contact the app author."
	case startupProfile:
		return "The application data folder could not be prepared.\n\nCheck access to your local application data folder and available disk space, then try again. Your existing data has not been reset."
	case startupRuntime:
		return "WebView2 could not be initialized.\n\nCheck that Microsoft Edge WebView2 Runtime is installed. Close other copies of this app and try again. If it still fails, contact the app author."
	case startupDisplay:
		return "The application's display scaling could not be initialized.\n\nCheck Windows compatibility settings for this app and remove any forced DPI override, then try again."
	case startupInstance:
		return "The application's single-instance lock could not be acquired.\n\nClose other copies of this app and check access to your application data folder, then try again."
	default:
		return "The application could not start.\n\nExtract the complete app package into a folder and try again. If it still fails, contact the app author."
	}
}

func showStartupFailure(kind startupFailureKind) {
	// Never interpolate configuration, paths, native error strings or document data.
	body, _ := windows.UTF16PtrFromString(startupFailureMessage(kind))
	caption, _ := windows.UTF16PtrFromString("Velox - Unable to start")
	windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(
		0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), 0x10|0x10000,
	)
	runtime.KeepAlive(body)
	runtime.KeepAlive(caption)
}
