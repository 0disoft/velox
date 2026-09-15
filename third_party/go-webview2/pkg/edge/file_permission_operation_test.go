//go:build windows

package edge

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

type permissionTestSetting struct {
	permissionSetting
	origin string
	kind   CoreWebView2PermissionKind
	state  CoreWebView2PermissionState
}
type permissionTestCollection struct {
	permissionCollection
	items []*permissionTestSetting
}
type permissionTestProfile struct {
	filePermissionProfile
	writes, reads, releases int
	origin                  string
	state                   CoreWebView2PermissionState
}

func permissionFixture(items ...*permissionTestSetting) *permissionTestCollection {
	collection := &permissionTestCollection{items: items}
	collection.vtbl = &struct {
		_IUnknownVtbl
		GetValueAtIndex, GetCount ComProc
	}{
		GetCount: NewComProc(func(c *permissionTestCollection, out *uint32) uintptr { *out = uint32(len(c.items)); return 0 }),
		GetValueAtIndex: NewComProc(func(c *permissionTestCollection, index uint32, out **permissionSetting) uintptr {
			*out = &c.items[index].permissionSetting
			return 0
		}),
	}
	for _, item := range items {
		item.vtbl = &struct {
			_IUnknownVtbl
			GetKind, GetOrigin, GetState ComProc
		}{
			_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(p *permissionTestSetting) uintptr { return 0 })},
			GetKind:       NewComProc(func(p *permissionTestSetting, out *CoreWebView2PermissionKind) uintptr { *out = p.kind; return 0 }),
			GetState:      NewComProc(func(p *permissionTestSetting, out *CoreWebView2PermissionState) uintptr { *out = p.state; return 0 }),
			GetOrigin: NewComProc(func(p *permissionTestSetting, out **uint16) uintptr {
				data, _ := windows.UTF16FromString(p.origin)
				memory, _, _ := windows.NewLazySystemDLL("ole32.dll").NewProc("CoTaskMemAlloc").Call(uintptr(len(data) * 2))
				if memory == 0 {
					return 0x8007000e
				}
				copy(unsafe.Slice((*uint16)(unsafe.Pointer(memory)), len(data)), data)
				*out = (*uint16)(unsafe.Pointer(memory))
				return 0
			}),
		}
	}
	return collection
}

func permissionProfileFixture() *permissionTestProfile {
	p := &permissionTestProfile{}
	p.vtbl = &filePermissionProfileVtbl{
		_IUnknownVtbl:                   _IUnknownVtbl{Release: NewComProc(func(p *permissionTestProfile) uintptr { p.releases++; return 0 })},
		GetNonDefaultPermissionSettings: NewComProc(func(p *permissionTestProfile, handler *filePermissionHandler) uintptr { p.reads++; return 0 }),
		SetPermissionState: NewComProc(func(p *permissionTestProfile, kind CoreWebView2PermissionKind, origin *uint16, state CoreWebView2PermissionState, handler *filePermissionHandler) uintptr {
			if kind != CoreWebView2PermissionKindFileReadWrite || state != CoreWebView2PermissionStateDefault {
				return 0x80070057
			}
			p.writes++
			p.origin = windows.UTF16PtrToString(origin)
			p.state = state
			return 0
		}),
	}
	return p
}

func TestFilePermissionResetOnlyChangesMatchedDenialAndVerifies(t *testing.T) {
	const origin = "https://test.app.invalid"
	for _, reset := range []bool{false, true} {
		for _, state := range []CoreWebView2PermissionState{0, 1, 2} {
			e := NewChromium()
			profile := permissionProfileFixture()
			completed := false
			e.filePermissionOperation = &filePermissionOperation{profile: &profile.filePermissionProfile, origin: origin, reset: reset, done: func(got CoreWebView2PermissionState, err error) {
				completed = true
				if err != nil {
					t.Fatal(err)
				}
			}}
			collection := permissionFixture(&permissionTestSetting{origin: "https://other.invalid", kind: 8, state: 2}, &permissionTestSetting{origin: origin, kind: 3, state: 2}, &permissionTestSetting{origin: origin + "/", kind: 8, state: state})
			e.filePermissionReadCompleted(0, &collection.permissionCollection)
			if reset && state == 2 {
				if completed || profile.writes != 1 || profile.origin != origin || profile.state != 0 {
					t.Fatal("reset must target only this denial and await completion")
				}
				e.filePermissionSetCompleted(0)
				if completed || profile.reads != 1 {
					t.Fatal("reset must await readback")
				}
				clean := permissionFixture()
				e.filePermissionReadCompleted(0, &clean.permissionCollection)
			} else if profile.writes != 0 {
				t.Fatal("non-denied or read-only request changed a permission")
			}
			if !completed || profile.releases != 1 || e.filePermissionOperation != nil {
				t.Fatal("operation did not finish and release the profile")
			}
		}
	}
}

func TestFilePermissionShutdownAndFailedReadDoNotReset(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		e := NewChromium()
		profile := permissionProfileFixture()
		calls := 0
		e.filePermissionOperation = &filePermissionOperation{profile: &profile.filePermissionProfile, reset: true, done: func(_ CoreWebView2PermissionState, err error) {
			calls++
			if err == nil {
				t.Fatal("missing read error")
			}
		}}
		if shutdown {
			e.Destroy()
		}
		e.filePermissionReadCompleted(0x80004005, nil)
		e.filePermissionSetCompleted(0)
		if profile.writes != 0 || profile.releases != 1 || (shutdown && calls != 0) || (!shutdown && calls != 1) {
			t.Fatal("failed or canceled operation changed state or leaked")
		}
	}
}

func TestFilePermissionRejectsUntrustedOrNonOriginInput(t *testing.T) {
	e := NewChromium()
	e.webview = &ICoreWebView2{}
	e.FileSystemAccessAllowed = func(origin string) bool { return origin != "https://other.invalid" }
	for _, origin := range []string{"http://test.invalid", "https://test.invalid/path", "https://user@test.invalid", "https://test.invalid?q=1", "https://test.invalid#fragment", "https://other.invalid"} {
		if e.FilePermission(origin, true, func(CoreWebView2PermissionState, error) {}) == nil {
			t.Fatalf("accepted %s", origin)
		}
	}
	e.webview = nil
}
