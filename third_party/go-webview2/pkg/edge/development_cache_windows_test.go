//go:build windows

package edge

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestDevelopmentCachePolicy(t *testing.T) {
	for _, test := range []struct {
		name        string
		debug       bool
		initialized bool
		destroyed   bool
		hresult     uintptr
		failAt      int
		wantCalls   int
		wantError   bool
	}{
		{name: "production"},
		{name: "production-initialized", initialized: true},
		{name: "debug-uninitialized", debug: true, wantError: true},
		{name: "debug-destroyed", debug: true, initialized: true, destroyed: true, wantError: true},
		{name: "debug", debug: true, initialized: true, wantCalls: 2},
		{name: "enable-rejected", debug: true, initialized: true, hresult: 0x80070057, failAt: 1, wantCalls: 1, wantError: true},
		{name: "cache-policy-rejected", debug: true, initialized: true, hresult: 0x80070057, failAt: 2, wantCalls: 2, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			e := &Chromium{destroyed: test.destroyed}
			if test.initialized {
				e.webview = &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{
					CallDevToolsProtocolMethod: NewComProc(func(_ uintptr, method, params *uint16, callback uintptr) uintptr {
						calls++
						wantMethod, wantParams := "Network.enable", `{}`
						if calls == 2 {
							wantMethod, wantParams = "Network.setCacheDisabled", `{"cacheDisabled":true}`
						}
						if windows.UTF16PtrToString(method) != wantMethod || windows.UTF16PtrToString(params) != wantParams || callback != 0 {
							t.Error("unexpected development protocol call")
						}
						if calls == test.failAt {
							return test.hresult
						}
						return 0
					}),
				}}
			}
			err := e.ConfigureDevelopmentCache(test.debug)
			if (err != nil) != test.wantError || calls != test.wantCalls {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}
