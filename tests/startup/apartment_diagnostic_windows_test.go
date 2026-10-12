package startup_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type apartmentProbeReceipt struct {
	Balanced       bool   `json:"balanced"`
	Fresh          bool   `json:"freshThread"`
	Before         string `json:"beforeHresult"`
	Init           string `json:"initializeHresult"`
	After          string `json:"afterHresult"`
	AfterType      uint32 `json:"afterApartmentType"`
	AfterQualifier uint32 `json:"afterApartmentQualifier"`
	ExitCode       int    `json:"hostExitCode"`
}

func parseApartmentProbeResult(output string) (*apartmentProbeReceipt, error) {
	var result *apartmentProbeReceipt
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "VELOX_APARTMENT_PROBE=") {
			continue
		}
		if result != nil {
			return nil, errors.New("duplicate apartment probe receipt")
		}
		var receipt apartmentProbeReceipt
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "VELOX_APARTMENT_PROBE="))), &receipt); err != nil {
			return nil, err
		}
		if receipt.Before == "" || receipt.After == "" || (receipt.Init != "0x00000000" && receipt.Init != "0x00000001") || receipt.ExitCode != 0 {
			return nil, errors.New("invalid apartment probe receipt")
		}
		result = &receipt
	}
	return result, nil
}

func TestApartmentProbeReceipt(t *testing.T) {
	valid := `VELOX_APARTMENT_PROBE={"balanced":true,"beforeHresult":"0x800401f0","initializeHresult":"0x00000000","afterHresult":"0x800401f0","hostExitCode":0}`
	if result, err := parseApartmentProbeResult("test output\n" + valid); err != nil || result == nil || !result.Balanced {
		t.Fatalf("valid receipt: %+v %v", result, err)
	}
	for _, invalid := range []string{valid + "\n" + valid, "VELOX_APARTMENT_PROBE={}", "VELOX_APARTMENT_PROBE=not-json", strings.Replace(valid, "0x00000000", "0x80010106", 1)} {
		if _, err := parseApartmentProbeResult(invalid); err == nil {
			t.Fatal("invalid receipt accepted")
		}
	}
	if result, err := parseApartmentProbeResult("ordinary host output"); err != nil || result != nil {
		t.Fatal("ordinary host output requires no test-only receipt")
	}
}

func TestApartmentReleaseDiagnostic(t *testing.T) {
	runApartmentReleaseDiagnostic(t, "VELOX_APARTMENT_DIAGNOSTIC_RESULT", false)
}

func TestFreshApartmentReleaseDiagnostic(t *testing.T) {
	runApartmentReleaseDiagnostic(t, "VELOX_FRESH_APARTMENT_DIAGNOSTIC_RESULT", true)
}

func runApartmentReleaseDiagnostic(t *testing.T, resultEnv string, fresh bool) {
	t.Helper()
	if os.Getenv(resultEnv) == "" {
		t.Skip("explicit result path required for test-only apartment comparison")
	}
	executable := requiredExecutable(t, "VELOX_APARTMENT_PROBE_HOST")
	config := filepath.Join(repositoryRoot(t), "examples", "hello", "velox.runtime.json")
	evidence := runProfileReleaseDiagnosticWithHost(t, resultEnv,
		[]string{"apartment-unbalanced", "apartment-balanced", "apartment-balanced", "apartment-unbalanced"}, func(mode string) hostAdapter {
			balanced := "0"
			if mode == "apartment-balanced" {
				balanced = "1"
			}
			return hostAdapter{name: mode, executable: executable,
				arguments: func(string) []string { return []string{"-test.run=^TestHostApartmentProbeChild$", "-test.v"} },
				environment: func(profile string) []string {
					freshThread := "0"
					if fresh {
						freshThread = "1"
					}
					return []string{"VELOX_DATA_DIR=" + profile, "VELOX_APARTMENT_PROBE_CONFIG=" + config, "VELOX_APARTMENT_PROBE_BALANCED=" + balanced, "VELOX_APARTMENT_PROBE_FRESH_THREAD=" + freshThread}
				}, expectedPhase: "dom-2raf", requireBrowserProcess: true}
		})
	for _, trial := range evidence.Trials {
		wantBalanced := trial.Mode == "apartment-balanced"
		for _, receipt := range []*apartmentProbeReceipt{trial.FirstApartment, trial.SecondApartment} {
			if receipt == nil || receipt.Balanced != wantBalanced || receipt.Fresh != fresh {
				t.Fatalf("comparison arm lacks matching receipt: %s %+v", trial.Mode, receipt)
			}
			if fresh {
				if receipt.Before != "0x800401f0" || receipt.Init != "0x00000000" {
					t.Fatalf("fresh STA ownership was not demonstrated: %+v", receipt)
				}
				if receipt.After != "0x00000000" || (!wantBalanced && (receipt.AfterType != 0 || receipt.AfterQualifier != 0)) ||
					(wantBalanced && (receipt.AfterType != 1 || receipt.AfterQualifier != 1)) {
					t.Fatalf("fresh STA release did not match the comparison arm: %+v", receipt)
				}
				// A successful query alone does not demonstrate an STA reference:
				// COM may report an implicit MTA after the owned STA is released.
				t.Logf("balanced=%t after=%s type=%d qualifier=%d", receipt.Balanced, receipt.After, receipt.AfterType, receipt.AfterQualifier)
			}
		}
		t.Logf("mode=%s apartment init=%s before=%s after=%s; second browser exit=%s", trial.Mode,
			trial.SecondApartment.Init, trial.SecondApartment.Before, trial.SecondApartment.After, diagnosticMilliseconds(trial.SecondBrowserExitMs))
	}
}
