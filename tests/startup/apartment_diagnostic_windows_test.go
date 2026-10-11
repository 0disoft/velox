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
	Balanced bool   `json:"balanced"`
	Before   string `json:"beforeHresult"`
	Init     string `json:"initializeHresult"`
	After    string `json:"afterHresult"`
	ExitCode int    `json:"hostExitCode"`
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
	if os.Getenv("VELOX_APARTMENT_DIAGNOSTIC_RESULT") == "" {
		t.Skip("explicit result path required for test-only apartment comparison")
	}
	executable := requiredExecutable(t, "VELOX_APARTMENT_PROBE_HOST")
	config := filepath.Join(repositoryRoot(t), "examples", "hello", "velox.runtime.json")
	evidence := runProfileReleaseDiagnosticWithHost(t, "VELOX_APARTMENT_DIAGNOSTIC_RESULT",
		[]string{"apartment-unbalanced", "apartment-balanced", "apartment-balanced", "apartment-unbalanced"}, func(mode string) hostAdapter {
			balanced := "0"
			if mode == "apartment-balanced" {
				balanced = "1"
			}
			return hostAdapter{name: mode, executable: executable,
				arguments: func(string) []string { return []string{"-test.run=^TestHostApartmentProbeChild$", "-test.v"} },
				environment: func(profile string) []string {
					return []string{"VELOX_DATA_DIR=" + profile, "VELOX_APARTMENT_PROBE_CONFIG=" + config, "VELOX_APARTMENT_PROBE_BALANCED=" + balanced}
				}, expectedPhase: "dom-2raf", requireBrowserProcess: true}
		})
	for _, trial := range evidence.Trials {
		wantBalanced := trial.Mode == "apartment-balanced"
		for _, receipt := range []*apartmentProbeReceipt{trial.FirstApartment, trial.SecondApartment} {
			if receipt == nil || receipt.Balanced != wantBalanced {
				t.Fatalf("comparison arm lacks matching receipt: %s %+v", trial.Mode, receipt)
			}
		}
		t.Logf("mode=%s apartment init=%s before=%s after=%s; second browser exit=%s", trial.Mode,
			trial.SecondApartment.Init, trial.SecondApartment.Before, trial.SecondApartment.After, diagnosticMilliseconds(trial.SecondBrowserExitMs))
	}
}
