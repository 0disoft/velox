package doctor

import (
	"errors"
	"testing"

	"github.com/0disoft/velox/internal/buildplan"
)

func TestEvaluateReportsUnavailableRuntimeWithoutClaimingReadiness(t *testing.T) {
	result, failure := Evaluate(Evidence{GOOS: "windows", GOARCH: "amd64", WindowsVersion: supportedClient(), PlanError: &buildplan.Error{Kind: buildplan.ErrorHost, Err: errors.New("missing")}})
	if result.Ready || failure == nil || failure.Code != "RUNTIME_WEBVIEW2_UNAVAILABLE" || failure.ExitCode != 5 {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
	if len(result.Checks) != 5 || result.Checks[1].Status != StatusPass || result.Checks[2].Status != StatusFail || result.Checks[3].Status != StatusPass || result.Checks[4].Status != StatusFail {
		t.Fatalf("unexpected checks: %+v", result.Checks)
	}
}

func TestEvaluatePreservesProjectFailureCategory(t *testing.T) {
	result, failure := Evaluate(Evidence{
		GOOS: "windows", GOARCH: "amd64", WindowsVersion: supportedClient(), WebView2Version: "123.0.0.0",
		PlanError: &buildplan.Error{Kind: buildplan.ErrorAsset, Err: errors.New("missing entry")},
	})
	if result.Ready || failure == nil || failure.Code != "ASSET_INVALID" || failure.ExitCode != 3 {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
	if result.Checks[3].Status != StatusFail || result.Checks[4].Status != StatusBlocked {
		t.Fatalf("unexpected project checks: %+v", result.Checks)
	}
}

func TestEvaluateRejectsUnsupportedPlatformEvenWithOtherEvidence(t *testing.T) {
	result, failure := Evaluate(Evidence{GOOS: "linux", GOARCH: "amd64", WebView2Version: "123.0.0.0", PlanError: &buildplan.Error{Kind: buildplan.ErrorHost, Err: errors.New("missing")}})
	if result.Ready || failure == nil || failure.Code != "RUNTIME_PLATFORM_UNSUPPORTED" {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
}

func TestEvaluateRejectsUnsupportedWindowsBuild(t *testing.T) {
	result, failure := Evaluate(Evidence{
		GOOS: "windows", GOARCH: "amd64",
		WindowsVersion:  WindowsVersion{Major: 10, Build: MinimumWindowsClientBuild - 1},
		WebView2Version: "123.0.0.0",
		PlanError:       &buildplan.Error{Kind: buildplan.ErrorHost, Err: errors.New("missing")},
	})
	if result.Ready || failure == nil || failure.Code != "RUNTIME_WINDOWS_VERSION_UNSUPPORTED" || result.Checks[1].Status != StatusFail {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
}

func TestEvaluateRejectsOldAndMalformedWebView2Versions(t *testing.T) {
	tests := []struct {
		version string
		code    string
	}{
		{version: "91.0.9999.9999", code: "RUNTIME_WEBVIEW2_UNSUPPORTED"},
		{version: "not-a-version", code: "RUNTIME_WEBVIEW2_VERSION_INVALID"},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			result, failure := Evaluate(Evidence{
				GOOS: "windows", GOARCH: "amd64", WindowsVersion: supportedClient(), WebView2Version: test.version,
				PlanError: &buildplan.Error{Kind: buildplan.ErrorHost, Err: errors.New("missing")},
			})
			if result.Ready || failure == nil || failure.Code != test.code || result.Checks[2].Status != StatusFail {
				t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
			}
		})
	}
}

func TestCompareBrowserVersionsAcceptsEvergreenChannelSuffix(t *testing.T) {
	comparison, err := compareBrowserVersions("92.0.902.49 stable", MinimumWebView2Version)
	if err != nil || comparison != 0 {
		t.Fatalf("comparison=%d error=%v", comparison, err)
	}
	comparison, err = compareBrowserVersions("123.0.0.0", MinimumWebView2Version)
	if err != nil || comparison <= 0 {
		t.Fatalf("comparison=%d error=%v", comparison, err)
	}
}

func TestSupportedWindowsBuildBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		version WindowsVersion
		want    bool
	}{
		{name: "client minimum", version: WindowsVersion{Major: 10, Build: MinimumWindowsClientBuild}, want: true},
		{name: "client below", version: WindowsVersion{Major: 10, Build: MinimumWindowsClientBuild - 1}, want: false},
		{name: "server minimum", version: WindowsVersion{Major: 10, Build: MinimumWindowsServerBuild, IsServer: true}, want: true},
		{name: "server below", version: WindowsVersion{Major: 10, Build: MinimumWindowsServerBuild - 1, IsServer: true}, want: false},
		{name: "old major", version: WindowsVersion{Major: 6, Build: 99999}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := supportedWindows(test.version); got != test.want {
				t.Fatalf("supportedWindows(%+v) = %v, want %v", test.version, got, test.want)
			}
		})
	}
}

func supportedClient() WindowsVersion {
	return WindowsVersion{Major: 10, Build: MinimumWindowsClientBuild}
}
