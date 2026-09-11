//go:build windows

package edge

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestFileSystemPermissionKeepsBrowserConsentAndDefaultDenials(t *testing.T) {
	const trusted = "https://app.velox.test/"
	const failed = uintptr(0x80004005)
	for _, tc := range []struct {
		name                                 string
		kind                                 CoreWebView2PermissionKind
		origin                               string
		gesture                              int32
		kindResult, gestureResult, uriResult uintptr
		disabled                             bool
		want                                 CoreWebView2PermissionState
	}{
		{name: "selected-file", kind: 8, origin: trusted, gesture: 1, want: CoreWebView2PermissionStateDefault},
		{name: "remote-origin", kind: 8, origin: "https://remote.test/", gesture: 1, want: CoreWebView2PermissionStateDeny},
		{name: "missing-origin", kind: 8, gesture: 1, want: CoreWebView2PermissionStateDeny},
		{name: "restored-file-handle", kind: 8, origin: trusted, want: CoreWebView2PermissionStateDefault},
		{name: "restored-foreign-handle", kind: 8, origin: "https://remote.test/", want: CoreWebView2PermissionStateDeny},
		{name: "kind-read-failed", kind: 8, origin: trusted, gesture: 1, kindResult: failed, want: CoreWebView2PermissionStateDeny},
		{name: "gesture-metadata-not-required", kind: 8, origin: trusted, gestureResult: failed, want: CoreWebView2PermissionStateDefault},
		{name: "origin-read-failed", kind: 8, origin: trusted, gesture: 1, uriResult: failed, want: CoreWebView2PermissionStateDeny},
		{name: "not-opted-in", kind: 8, origin: trusted, gesture: 1, disabled: true, want: CoreWebView2PermissionStateDeny},
		{name: "microphone", kind: 1, origin: trusted, gesture: 1, want: CoreWebView2PermissionStateDeny},
		{name: "camera", kind: 2, origin: trusted, gesture: 1, want: CoreWebView2PermissionStateDeny},
		{name: "clipboard", kind: 6, origin: trusted, gesture: 1, want: CoreWebView2PermissionStateDeny},
		{name: "downloads", kind: 7, origin: trusted, gesture: 1, want: CoreWebView2PermissionStateDeny},
		{name: "future-permission", kind: 99, origin: trusted, gesture: 1, want: CoreWebView2PermissionStateDeny},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewChromium()
			t.Cleanup(e.Destroy)
			e.SetGlobalPermission(CoreWebView2PermissionStateDeny)
			if !tc.disabled {
				e.FileSystemAccessAllowed = func(origin string) bool { return origin == trusted }
			}
			blocked, writes := 0, 0
			e.PolicyBlocked = func(kind string) {
				if kind == "permission" {
					blocked++
				}
			}
			var state CoreWebView2PermissionState
			args := &iCoreWebView2PermissionRequestedEventArgs{vtbl: &iCoreWebView2PermissionRequestedEventArgsVtbl{
				GetPermissionKind:  NewComProc(func(_ uintptr, out *CoreWebView2PermissionKind) uintptr { *out = tc.kind; return tc.kindResult }),
				GetIsUserInitiated: NewComProc(func(_ uintptr, out *int32) uintptr { *out = tc.gesture; return tc.gestureResult }),
				GetURI: NewComProc(func(_ uintptr, out **uint16) uintptr {
					if tc.uriResult != 0 || tc.origin == "" {
						return tc.uriResult
					}
					value, err := windows.UTF16FromString(tc.origin)
					if err != nil {
						return failed
					}
					ptr, _, _ := windows.NewLazySystemDLL("ole32.dll").NewProc("CoTaskMemAlloc").Call(uintptr(len(value) * 2))
					if ptr == 0 {
						return failed
					}
					*out = (*uint16)(unsafe.Pointer(ptr))
					copy(unsafe.Slice(*out, len(value)), value)
					return 0
				}),
				PutState: NewComProc(func(_ uintptr, value CoreWebView2PermissionState) uintptr { writes++; state = value; return 0 }),
			}}
			e.PermissionRequested(nil, args)
			if writes != 1 || state != tc.want {
				t.Fatalf("writes=%d state=%d want=%d", writes, state, tc.want)
			}
			if (blocked == 1) != (tc.want == CoreWebView2PermissionStateDeny) {
				t.Fatalf("blocked=%d state=%d", blocked, state)
			}
		})
	}
}
