package doctor

import (
	"errors"
	"testing"

	"github.com/0disoft/velox/internal/buildplan"
)

func TestEvaluateReportsUnavailableRuntimeWithoutClaimingReadiness(t *testing.T) {
	result, failure := Evaluate(Evidence{GOOS: "windows", GOARCH: "amd64", PlanError: &buildplan.Error{Kind: buildplan.ErrorHost, Err: errors.New("missing")}})
	if result.Ready || failure == nil || failure.Code != "RUNTIME_WEBVIEW2_UNAVAILABLE" || failure.ExitCode != 5 {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
	if len(result.Checks) != 4 || result.Checks[1].Status != StatusFail || result.Checks[2].Status != StatusPass || result.Checks[3].Status != StatusFail {
		t.Fatalf("unexpected checks: %+v", result.Checks)
	}
}

func TestEvaluatePreservesProjectFailureCategory(t *testing.T) {
	result, failure := Evaluate(Evidence{
		GOOS: "windows", GOARCH: "amd64", WebView2Version: "1.0.0",
		PlanError: &buildplan.Error{Kind: buildplan.ErrorAsset, Err: errors.New("missing entry")},
	})
	if result.Ready || failure == nil || failure.Code != "ASSET_INVALID" || failure.ExitCode != 3 {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
	if result.Checks[2].Status != StatusFail || result.Checks[3].Status != StatusBlocked {
		t.Fatalf("unexpected project checks: %+v", result.Checks)
	}
}

func TestEvaluateRejectsUnsupportedPlatformEvenWithOtherEvidence(t *testing.T) {
	result, failure := Evaluate(Evidence{GOOS: "linux", GOARCH: "amd64", WebView2Version: "1.0.0", PlanError: &buildplan.Error{Kind: buildplan.ErrorHost, Err: errors.New("missing")}})
	if result.Ready || failure == nil || failure.Code != "RUNTIME_PLATFORM_UNSUPPORTED" {
		t.Fatalf("unexpected evaluation: result=%+v failure=%+v", result, failure)
	}
}
