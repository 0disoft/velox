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
	Balanced       bool    `json:"balanced"`
	Fresh          bool    `json:"freshThread"`
	Before         string  `json:"beforeHresult"`
	Init           string  `json:"initializeHresult"`
	After          string  `json:"afterHresult"`
	AfterType      uint32  `json:"afterApartmentType"`
	AfterQualifier uint32  `json:"afterApartmentQualifier"`
	ExitCode       int     `json:"hostExitCode"`
	PostRun        string  `json:"postRun,omitempty"`
	PostRunMS      float64 `json:"postRunMs,omitempty"`
	Messages       int     `json:"postRunMessages,omitempty"`
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
				if !validFreshApartmentRelease(receipt) {
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

func validFreshApartmentRelease(receipt *apartmentProbeReceipt) bool {
	if receipt.Balanced {
		return receipt.After == "0x800401f0" || (receipt.After == "0x00000000" && receipt.AfterType == 1 && receipt.AfterQualifier == 1)
	}
	return receipt.After == "0x00000000" && (receipt.AfterType == 0 || receipt.AfterType == 3) && receipt.AfterQualifier == 0
}

func TestFreshApartmentReleaseClassification(t *testing.T) {
	for _, tc := range []struct {
		name    string
		receipt apartmentProbeReceipt
		want    bool
	}{
		{"released-uninitialized", apartmentProbeReceipt{Balanced: true, After: "0x800401f0"}, true},
		{"released-implicit-mta", apartmentProbeReceipt{Balanced: true, After: "0x00000000", AfterType: 1, AfterQualifier: 1}, true},
		{"released-still-sta", apartmentProbeReceipt{Balanced: true, After: "0x00000000"}, false},
		{"released-explicit-mta", apartmentProbeReceipt{Balanced: true, After: "0x00000000", AfterType: 1}, false},
		{"owned-sta", apartmentProbeReceipt{After: "0x00000000"}, true},
		{"owned-main-sta", apartmentProbeReceipt{After: "0x00000000", AfterType: 3}, true},
		{"owned-not-initialized", apartmentProbeReceipt{After: "0x800401f0"}, false},
		{"owned-implicit-mta", apartmentProbeReceipt{After: "0x00000000", AfterType: 1, AfterQualifier: 1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validFreshApartmentRelease(&tc.receipt); got != tc.want {
				t.Fatalf("classified=%t want=%t: %+v", got, tc.want, tc.receipt)
			}
		})
	}
}

func TestPostRunPumpDiagnostic(t *testing.T) {
	if os.Getenv("VELOX_POST_RUN_DIAGNOSTIC_RESULT") == "" {
		t.Skip("explicit result path required for test-only post-loop comparison")
	}
	executable := requiredExecutable(t, "VELOX_APARTMENT_PROBE_HOST")
	config := filepath.Join(repositoryRoot(t), "examples", "hello", "velox.runtime.json")
	evidence := runProfileReleaseDiagnosticWithHost(t, "VELOX_POST_RUN_DIAGNOSTIC_RESULT",
		[]string{"idle", "pump", "pump", "idle"}, func(mode string) hostAdapter {
			return hostAdapter{name: mode, executable: executable,
				arguments: func(string) []string { return []string{"-test.run=^TestHostApartmentProbeChild$", "-test.v"} },
				environment: func(profile string) []string {
					return []string{"VELOX_DATA_DIR=" + profile, "VELOX_APARTMENT_PROBE_CONFIG=" + config,
						"VELOX_APARTMENT_PROBE_BALANCED=1", "VELOX_APARTMENT_PROBE_FRESH_THREAD=1", "VELOX_APARTMENT_PROBE_POST_RUN=" + mode}
				}, expectedPhase: "dom-2raf", requireBrowserProcess: true}
		})
	for _, trial := range evidence.Trials {
		for _, receipt := range []*apartmentProbeReceipt{trial.FirstApartment, trial.SecondApartment} {
			if receipt == nil || !receipt.Balanced || !receipt.Fresh || receipt.PostRun != trial.Mode || receipt.PostRunMS < 990 ||
				receipt.Before != "0x800401f0" || receipt.Init != "0x00000000" || !validFreshApartmentRelease(receipt) {
				t.Fatalf("post-loop arm lacks matching fresh STA receipt: %s %+v", trial.Mode, receipt)
			}
		}
		t.Logf("mode=%s messages=%d/%d; second browser exit=%s", trial.Mode,
			trial.FirstApartment.Messages, trial.SecondApartment.Messages, diagnosticMilliseconds(trial.SecondBrowserExitMs))
	}
}
