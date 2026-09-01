//go:build windows

package evalsandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	probeEnabledEnv   = "VELOX_EVAL_SANDBOX_PROBE"
	probeAllowedEnv   = "VELOX_EVAL_SANDBOX_ALLOWED"
	probeForbiddenEnv = "VELOX_EVAL_SANDBOX_FORBIDDEN"
)

func TestAppContainerAndJobObjectEnforceEvaluationBoundary(t *testing.T) {
	base, err := os.MkdirTemp(".", ".velox-eval-sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	base, err = filepath.Abs(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("remove sandbox test root: %v", err)
		}
	})
	trialRoot := filepath.Join(base, "trial")
	forbiddenRoot := filepath.Join(base, "forbidden")
	receiptRoot := filepath.Join(base, "receipts")
	for _, directory := range []string{trialRoot, forbiddenRoot, receiptRoot} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	forbiddenPath := filepath.Join(forbiddenRoot, "sentinel.txt")
	if err := os.WriteFile(forbiddenPath, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	testExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	toolRoot := filepath.Join(base, "tool")
	if err := os.MkdirAll(toolRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(toolRoot, "velox-eval-sandbox-test.exe")
	executableBody, err := os.ReadFile(testExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, executableBody, 0o700); err != nil {
		t.Fatal(err)
	}
	allowedPath := filepath.Join(trialRoot, "inside.txt")
	prompt := "run the sandbox boundary probe"
	promptPath := filepath.Join(receiptRoot, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte(prompt), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(probeEnabledEnv, "1")
	t.Setenv(probeAllowedEnv, allowedPath)
	t.Setenv(probeForbiddenEnv, forbiddenPath)
	receiptPath := filepath.Join(receiptRoot, "receipt.json")
	stateDatabaseExportPath := filepath.Join(receiptRoot, "state.db")
	receipt, err := Run(Config{
		TrialID:                 "trial-20260816T010203Z-a1b2c3d4",
		SeriesID:                "series-20260816T010203Z-a1b2c3d4",
		Sequence:                1,
		TrialRoot:               trialRoot,
		ToolRoots:               []string{toolRoot},
		PassEnvironment:         []string{probeEnabledEnv, probeAllowedEnv, probeForbiddenEnv},
		PromptPath:              promptPath,
		StateDatabaseExportPath: stateDatabaseExportPath,
		ReceiptPath:             receiptPath,
		Timeout:                 30 * time.Second,
		Command:                 []string{executable, "-test.run=^TestSandboxBoundaryProbeProcess$", "--", prompt},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Containment.FilesystemEnforced || !receipt.Containment.ProcessTreeEnforced || !receipt.Containment.CleanupCompleted {
		t.Fatalf("incomplete containment receipt: %#v", receipt.Containment)
	}
	if receipt.PromptSHA256 != digest([]byte(prompt)) {
		t.Fatal("sandbox receipt did not bind the evaluator prompt")
	}
	if _, err := os.Stat(receiptPath); err != nil {
		t.Fatalf("sandbox receipt missing: %v", err)
	}
	if body, err := os.ReadFile(stateDatabaseExportPath); err != nil || string(body) != "isolated-state" {
		t.Fatalf("isolated state database export missing: %q %v", body, err)
	}
	if body, err := os.ReadFile(allowedPath); err != nil || string(body) != "inside\ngrandchild" {
		t.Fatalf("allowed sandbox writes missing: %q %v", body, err)
	}
	if body, err := os.ReadFile(forbiddenPath); err != nil || string(body) != "outside" {
		t.Fatalf("forbidden sentinel changed: %q %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(forbiddenRoot, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("sandbox wrote outside the trial root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(trialRoot, ".velox-sandbox")); !os.IsNotExist(err) {
		t.Fatalf("private sandbox environment was not removed: %v", err)
	}
}

func TestSandboxBoundaryProbeProcess(t *testing.T) {
	if os.Getenv(probeEnabledEnv) != "1" {
		return
	}
	forbidden := os.Getenv(probeForbiddenEnv)
	if _, err := os.ReadFile(forbidden); err == nil {
		t.Fatal("AppContainer read a file outside its explicit grants")
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(forbidden), "escaped.txt"), []byte("escape"), 0o600); err == nil {
		t.Fatal("AppContainer wrote outside its explicit grants")
	}
	if err := os.WriteFile(os.Getenv(probeAllowedEnv), []byte("inside"), 0o600); err != nil {
		t.Fatalf("AppContainer could not write inside the trial root: %v", err)
	}
	hermesHome := os.Getenv("HERMES_HOME")
	if err := os.MkdirAll(hermesHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hermesHome, "state.db"), []byte("isolated-state"), 0o600); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestSandboxGrandchildProbeProcess$")
	child.Env = os.Environ()
	child.Stdin = strings.NewReader("")
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("contained grandchild failed: %v: %s", err, output)
	}
}

func TestSandboxGrandchildProbeProcess(t *testing.T) {
	if os.Getenv(probeEnabledEnv) != "1" {
		return
	}
	file, err := os.OpenFile(os.Getenv(probeAllowedEnv), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("\ngrandchild"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}
