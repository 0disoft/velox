package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeCancellationWorkflowContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "native-cancellation.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"workflow_dispatch:", "runs-on: windows-2025", "timeout-minutes: 10", "contents: read", "persist-credentials: false",
		"cancel-in-progress: false", "cache: false", "hosted-source-fork", "measurementCommit = $env:GITHUB_SHA",
		"githubRunId = $env:GITHUB_RUN_ID", "imageVersion = $env:ImageVersion", "requestedCases = 6",
		"publicReleaseTested = $false", "lateEnvironmentCompletionTested = $false", "VELOX_NATIVE_CANCELLATION = '1'",
		"go test -mod=readonly -json -count=1 -timeout=180s", "'^TestNativeInitializationCancellation$'",
		"go-test.jsonl", "go-test.stderr.txt", "ConvertFrom-Json", "'environment-completion', 'controller-pending'",
		"$terminal.Count -eq 1", "$testExit -eq 0", "$parentPass.Count -eq 1", "$runtime.Count -eq 1", "$cases.Count -eq 6",
		"Where-Object outcome -CNE 'pass'", "source-fork evidence:", "if (-not $passed) { throw", "if: always()", "retention-days: 90",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("missing native cancellation contract %q", required)
		}
	}
	for _, forbidden := range []string{"schedule:", "pull_request:", "push:", "contents: write", "continue-on-error:", "TestStartupLifecycleEvidence", "hermes", "go build", "go run"} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("unexpected workflow surface %q", forbidden)
		}
	}
}
