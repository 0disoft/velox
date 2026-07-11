package hygiene_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestConsumerEvidenceWorkflowPinsActionsAndAvoidsCaches(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "consumer-evidence.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)

	if strings.Contains(workflow, "actions/cache@") {
		t.Fatal("consumer evidence workflow must not use actions/cache")
	}
	if !strings.Contains(workflow, "runs-on: windows-2025") {
		t.Fatal("consumer evidence workflow must pin the Windows runner label")
	}
	if !strings.Contains(workflow, "retention-days: 1") || !strings.Contains(workflow, "retention-days: 7") {
		t.Fatal("consumer evidence workflow must bound release and raw-result retention")
	}

	usesPattern := regexp.MustCompile(`(?m)^\s*uses:\s+([^\s#]+)`)
	shaPattern := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}$`)
	matches := usesPattern.FindAllStringSubmatch(workflow, -1)
	if len(matches) == 0 {
		t.Fatal("consumer evidence workflow contains no actions")
	}
	for _, match := range matches {
		if !shaPattern.MatchString(match[1]) {
			t.Errorf("workflow action is not pinned to an immutable SHA: %s", match[1])
		}
	}
}

func TestConsumerEvidenceWorkflowKeepsConsumerBuildCompilerFree(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "consumer-evidence.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	consumerIndex := strings.Index(workflow, "\n  consumer:\n")
	if consumerIndex < 0 {
		t.Fatal("consumer job is missing")
	}
	consumer := workflow[consumerIndex:]
	for _, forbidden := range []string{"go build", "go run", "go test", "setup-go", "actions/cache"} {
		if strings.Contains(consumer, forbidden) {
			t.Errorf("consumer job contains forbidden toolchain surface %q", forbidden)
		}
	}
	if !strings.Contains(consumer, "scripts/measure-consumer-e2e.ps1") {
		t.Fatal("consumer job does not invoke the end-to-end measurement contract")
	}
}
