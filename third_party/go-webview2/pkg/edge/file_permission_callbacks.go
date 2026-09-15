//go:build windows

package edge

import "unsafe"

var filePermissionReadIID = NewGUID("38274481-A15C-4563-94CF-990EDC9AEB95")
var filePermissionSetIID = NewGUID("FC77FB30-9C9E-4076-B8C7-7644A703CA1B")

type filePermissionHandler struct {
	vtbl *struct {
		_IUnknownVtbl
		Invoke ComProc
	}
	impl *Chromium
	iid  *GUID
}

func filePermissionQuery(h *filePermissionHandler, iid *GUID, out *unsafe.Pointer) uintptr {
	return queryCallbackInterface(h.impl, unsafe.Pointer(h), iid, out, h.iid)
}
func filePermissionAddRef(h *filePermissionHandler) uintptr  { return h.impl.AddRef() }
func filePermissionRelease(h *filePermissionHandler) uintptr { return h.impl.Release() }
func filePermissionReadInvoke(h *filePermissionHandler, status uintptr, collection *permissionCollection) uintptr {
	h.impl.filePermissionReadCompleted(status, collection)
	return 0
}
func filePermissionSetInvoke(h *filePermissionHandler, status uintptr) uintptr {
	h.impl.filePermissionSetCompleted(status)
	return 0
}

var filePermissionReadVtbl = struct {
	_IUnknownVtbl
	Invoke ComProc
}{
	_IUnknownVtbl{NewComProc(filePermissionQuery), NewComProc(filePermissionAddRef), NewComProc(filePermissionRelease)}, NewComProc(filePermissionReadInvoke),
}
var filePermissionSetVtbl = struct {
	_IUnknownVtbl
	Invoke ComProc
}{
	_IUnknownVtbl{NewComProc(filePermissionQuery), NewComProc(filePermissionAddRef), NewComProc(filePermissionRelease)}, NewComProc(filePermissionSetInvoke),
}
