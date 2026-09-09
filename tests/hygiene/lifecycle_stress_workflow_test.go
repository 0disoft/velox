package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleStressWorkflowContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "lifecycle-stress.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"workflow_dispatch:", "runs-on: windows-2025", "timeout-minutes: 15",
		"persist-credentials: false", "contents: read", "cancel-in-progress: false",
		"v0.5.10-alpha.40", "771173b6eec2f74d92228e7ac5b52332160b0f9fb4d7baecf01976864a00f8c8",
		"$observed -cne $expected", "hostSha256", "measurementCommit",
		"initializationCancellationTested = $false", "requestedLaunches = 100",
		"VELOX_STARTUP_LIFECYCLE_REPETITIONS = '50'", "-timeout=12m",
		"'^TestStartupLifecycleEvidence$'", "if ($testExit -ne 0) { exit $testExit }",
		"Test-Json -Schema $schema", "$result.samples.Count -ne 50",
		"Where-Object outcome -NE 'success'", "hosted-runner-evidence",
		"if: always()", "retention-days: 90",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("missing stress contract %q", required)
		}
	}
	for _, forbidden := range []string{"go build", "go run", "contents: write", "schedule:", "continue-on-error:"} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("unexpected stress workflow surface %q", forbidden)
		}
	}
}
