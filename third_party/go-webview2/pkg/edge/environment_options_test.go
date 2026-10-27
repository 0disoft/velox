//go:build windows

package edge

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestLegacyReputationVersionBoundary(t *testing.T) {
	for _, test := range []struct {
		version string
		want    bool
	}{
		{"91.0.864.0", false}, {"92.0.902.49", true}, {"111.0.1661.62", true},
		{"111.0.1661.62 beta", true}, {"112.0.1722.0", false}, {"155.0.4283.62", false},
		{"", false}, {"111", false}, {"111.x.0.0", false}, {"111.0.0.-1", false},
		{"111.0.0.4294967296", false},
	} {
		if got := needsLegacyReputationOption(test.version); got != test.want {
			t.Fatalf("legacy reputation gate for %q: %v, want %v", test.version, got, test.want)
		}
	}
}

func TestLegacyReputationRequiresExplicitLocalContentPolicy(t *testing.T) {
	for missing := 0; missing < 5; missing++ {
		e := NewChromium()
		e.DisableReputationChecking = missing != 0
		if missing != 1 {
			e.NavigationAllowed = func(string) bool { return false }
		}
		e.DenyFrames, e.DenyNewWindows, e.DenyDownloads = missing != 2, missing != 3, missing != 4
		if e.legacyReputationOptions() != nil || e.environmentOptions.strings[0] != "" {
			t.Fatalf("missing local-content guard %d still enabled the legacy flag", missing)
		}
	}
}

func TestEnvironmentOptionsCOMABIAndNativeStringOwnership(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	defer e.Destroy()
	o := e.environmentOptions
	if unsafe.Sizeof(environmentOptionsVtbl{}) != 11*unsafe.Sizeof(ComProc(0)) {
		t.Fatal("environment options must have three IUnknown and eight property slots")
	}
	for _, property := range []struct {
		get, put ComProc
		value    string
	}{
		{o.vtbl.GetArguments, o.vtbl.PutArguments, legacyReputationArgument},
		{o.vtbl.GetLanguage, o.vtbl.PutLanguage, "ko-KR"},
		{o.vtbl.GetTargetVersion, o.vtbl.PutTargetVersion, "111.0.1661.62"},
	} {
		input, _ := windows.UTF16PtrFromString(property.value)
		hr, _, _ := property.put.Call(uintptr(unsafe.Pointer(o)), uintptr(unsafe.Pointer(input)))
		if hr != 0 {
			t.Fatalf("put property: %x", hr)
		}
		var out *uint16
		hr, _, _ = property.get.Call(uintptr(unsafe.Pointer(o)), uintptr(unsafe.Pointer(&out)))
		if hr != 0 || out == nil {
			t.Fatalf("get property: %x", hr)
		}
		got := windows.UTF16PtrToString(out)
		windows.CoTaskMemFree(unsafe.Pointer(out))
		if got != property.value {
			t.Fatalf("property = %q, want %q", got, property.value)
		}
		property.put.Call(uintptr(unsafe.Pointer(o)), 0)
		out = input
		hr, _, _ = property.get.Call(uintptr(unsafe.Pointer(o)), uintptr(unsafe.Pointer(&out)))
		if hr != 0 || out != nil {
			t.Fatal("reset property retained stale output")
		}
		hr, _, _ = property.get.Call(uintptr(unsafe.Pointer(o)), 0)
		if hr != 0x80004003 {
			t.Fatal("missing string output accepted")
		}
	}
	state := struct {
		value int32
		guard uint32
	}{0, 0xdeadbeef}
	o.vtbl.PutSSO.Call(uintptr(unsafe.Pointer(o)), 99)
	hr, _, _ := o.vtbl.GetSSO.Call(uintptr(unsafe.Pointer(o)), uintptr(unsafe.Pointer(&state.value)))
	if hr != 0 || state.value != 1 || state.guard != 0xdeadbeef {
		t.Fatal("BOOL getter must write exactly four bytes and normalize true")
	}
	o.vtbl.PutSSO.Call(uintptr(unsafe.Pointer(o)), 0)
	o.vtbl.GetSSO.Call(uintptr(unsafe.Pointer(o)), uintptr(unsafe.Pointer(&state.value)))
	if state.value != 0 {
		t.Fatal("SSO did not reset")
	}
	hr, _, _ = o.vtbl.GetSSO.Call(uintptr(unsafe.Pointer(o)), 0)
	if hr != 0x80004003 {
		t.Fatal("missing BOOL output accepted")
	}
}
