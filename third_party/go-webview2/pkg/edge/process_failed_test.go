//go:build windows

package edge

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestProcessFailureOnlyMarksMainBrowserExit(t *testing.T) {
	for _, kind := range []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 999} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			e := NewChromium()
			e.retainCallbackOwner()
			defer e.Destroy()
			calls := 0
			e.BrowserProcessExitedCallback = func() { calls++ }
			args := &processFailedArgs{vtbl: &processFailedArgsVtbl{
				GetProcessFailedKind: NewComProc(func(_ uintptr, out *uint32) uintptr { *out = kind; return 0 }),
			}}
			for i := 0; i < 2; i++ {
				e.processFailed.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.processFailed)), 0, uintptr(unsafe.Pointer(args)))
			}
			want := 0
			if kind == browserProcessExitedKind {
				want = 1
			}
			if calls != want || e.browserProcessExited != (want == 1) {
				t.Fatalf("kind %d bypassed browser failure policy: calls=%d failed=%v", kind, calls, e.browserProcessExited)
			}
		})
	}
}

func TestProcessFailureRejectsUnreadableAndLateEvents(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	defer e.Destroy()
	calls := 0
	e.BrowserProcessExitedCallback = func() { calls++ }
	args := &processFailedArgs{vtbl: &processFailedArgsVtbl{
		GetProcessFailedKind: NewComProc(func(_ uintptr, out *uint32) uintptr { *out = 0; return 0x80004005 }),
	}}
	e.processFailed.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.processFailed)), 0, 0)
	e.processFailed.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.processFailed)), 0, uintptr(unsafe.Pointer(args)))
	e.AddRef()
	e.Destroy()
	args.vtbl.GetProcessFailedKind = NewComProc(func(_ uintptr, out *uint32) uintptr { *out = 0; return 0 })
	e.processFailed.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.processFailed)), 0, uintptr(unsafe.Pointer(args)))
	e.Release()
	if calls != 0 || e.browserProcessExited {
		t.Fatal("failed read or late callback marked browser failure")
	}
}
