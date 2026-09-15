//go:build windows

package edge

import (
	"errors"
	"net/url"
	"unsafe"
)

type filePermissionOperation struct {
	profile                   *filePermissionProfile
	origin                    string
	reset, setting, verifying bool
	done                      func(CoreWebView2PermissionState, error)
}

// FilePermission reads or resets only an allowed document's persisted
// FileReadWrite denial. Call on the UI thread after explicit native consent.
// Reset never grants access and leaves existing Default/Allow settings alone.
func (e *Chromium) FilePermission(origin string, reset bool, done func(CoreWebView2PermissionState, error)) error {
	if e.destroyed || e.webview == nil || done == nil {
		return errors.New("WebView2 is unavailable")
	}
	if e.filePermissionOperation != nil {
		return errors.New("file permission request is already pending")
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || e.FileSystemAccessAllowed == nil || !e.FileSystemAccessAllowed(origin) {
		return errors.New("file permission origin is not allowed")
	}
	profile, err := getFilePermissionProfile(e.webview)
	if err != nil {
		return err
	}
	e.filePermissionOperation = &filePermissionOperation{profile: profile, origin: origin, reset: reset, done: done}
	if err := profile.read(e.filePermissionRead); err != nil {
		e.finishFilePermission(0, err, false)
		return err
	}
	return nil
}

func (e *Chromium) finishFilePermission(state CoreWebView2PermissionState, err error, notify bool) {
	op := e.filePermissionOperation
	if op == nil {
		return
	}
	e.filePermissionOperation = nil
	releaseIUnknown(uintptr(unsafe.Pointer(op.profile)))
	if notify && !e.destroyed {
		op.done(state, err)
	}
}

func (e *Chromium) filePermissionReadCompleted(status uintptr, collection *permissionCollection) {
	op := e.filePermissionOperation
	if op == nil || e.destroyed || op.setting {
		return
	}
	if err := hresult(status); err != nil {
		e.finishFilePermission(0, err, true)
		return
	}
	state, err := readFilePermission(collection, op.origin)
	if err != nil {
		e.finishFilePermission(0, err, true)
		return
	}
	if op.verifying {
		if state != CoreWebView2PermissionStateDefault {
			err = errors.New("file permission reset was not confirmed")
		}
		e.finishFilePermission(state, err, true)
		return
	}
	if !op.reset || state != CoreWebView2PermissionStateDeny {
		e.finishFilePermission(state, nil, true)
		return
	}
	op.setting = true
	if err := op.profile.reset(op.origin, e.filePermissionSet); err != nil {
		e.finishFilePermission(0, err, true)
	}
}

func (e *Chromium) filePermissionSetCompleted(status uintptr) {
	op := e.filePermissionOperation
	if op == nil || e.destroyed || !op.setting {
		return
	}
	if err := hresult(status); err != nil {
		e.finishFilePermission(0, err, true)
		return
	}
	op.setting, op.verifying = false, true
	if err := op.profile.read(e.filePermissionRead); err != nil {
		e.finishFilePermission(0, err, true)
	}
}
