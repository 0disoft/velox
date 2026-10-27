//go:build windows

package edge

import (
	"testing"
	"unsafe"
)

func TestReputationSettingsHRESULTAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		query, setter            uintptr
		empty                    bool
		wantSupported, wantError bool
		wantWrites, wantReleases int
	}{
		{name: "supported", wantSupported: true, wantWrites: 1, wantReleases: 1},
		{name: "old runtime", query: 0x80004002},
		{name: "query failure", query: 0x80004005, wantError: true},
		{name: "missing interface", empty: true, wantError: true},
		{name: "setter failure", setter: 0x80004005, wantSupported: true, wantError: true, wantWrites: 1, wantReleases: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes, releases := 0, 0
			settings8 := &iCoreWebView2Settings8{vtbl: &iCoreWebView2Settings8Vtbl{}}
			settings8.vtbl.Release = NewComProc(func(uintptr) uintptr { releases++; return 0 })
			settings8.vtbl.PutIsReputationCheckingRequired = NewComProc(func(_ uintptr, required uintptr) uintptr {
				writes++
				if required != 0 {
					t.Error("local content enabled reputation checking")
				}
				return tc.setter
			})
			settings := &ICoreWebViewSettings{vtbl: &_ICoreWebViewSettingsVtbl{}}
			settings.vtbl.QueryInterface = NewComProc(func(_ uintptr, _ uintptr, out **iCoreWebView2Settings8) uintptr {
				if tc.query == 0 && !tc.empty {
					*out = settings8
				}
				return tc.query
			})
			supported, err := settings.SetReputationCheckingRequired(false)
			if supported != tc.wantSupported || (err != nil) != tc.wantError || writes != tc.wantWrites || releases != tc.wantReleases {
				t.Fatalf("supported=%v err=%v writes=%d releases=%d", supported, err, writes, releases)
			}
		})
	}
}

func TestReputationSettingsMatchesSDKSlots(t *testing.T) {
	// Official Settings8 C ABI: 3 IUnknown + 18 Settings1 + 2 Settings2 +
	// 2 Settings3 + 4 Settings4 + 2 Settings5 + 2 Settings6 + 2 Settings7.
	var table iCoreWebView2Settings8Vtbl
	if got := unsafe.Offsetof(table.PutIsReputationCheckingRequired) / unsafe.Sizeof(ComProc(0)); got != 36 {
		t.Fatalf("reputation setter COM slot=%d, want 36", got)
	}
}
