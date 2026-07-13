package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSummarizePreservesFailuresAndComputesOrdering(t *testing.T) {
	profile := 30.0
	runID, runAttempt := "123", "2"
	raw := evidence{
		SchemaVersion: "velox.startup-lifecycle/v2",
		Scope:         "fresh-and-immediate-same-profile-startup",
		EvidenceLevel: "controlled-local-observation",
		Outcome:       "failure",
		Repetitions:   3,
		Environment:   environment{OS: "windows", Architecture: "amd64", WebView2Version: "1.2.3", GitHubRunID: &runID, GitHubRunAttempt: &runAttempt},
		Samples: []sample{
			{Index: 0, Outcome: "success", First: &launch{ReadyMs: 100, BrowserExitAfterHostMs: 500}, Immediate: &launch{ReadyMs: 700, BrowserExitAfterHostMs: 600}, ProfileReleaseMs: &profile, Timeline: &timeline{FirstBrowserExitAfterImmediateStartMs: 500, ImmediateReadyAfterFirstBrowserExitMs: 200, ImmediateReadyWaitedForFirstBrowserExit: true}},
			{Index: 1, Outcome: "success", First: &launch{ReadyMs: 200, BrowserExitAfterHostMs: 600}, Immediate: &launch{ReadyMs: 800, BrowserExitAfterHostMs: 700}, ProfileReleaseMs: &profile, Timeline: &timeline{FirstBrowserExitAfterImmediateStartMs: 600, ImmediateReadyAfterFirstBrowserExitMs: -10, ImmediateReadyWaitedForFirstBrowserExit: false}},
			{Index: 2, Outcome: "failure", Error: &runError{Phase: "profile-release", Code: "PROFILE_RELEASE_FAILED"}},
		},
	}

	result, err := summarize(raw, []byte("source"))
	if err != nil {
		t.Fatal(err)
	}
	if result.SuccessCount != 2 || result.FailureCount != 1 || len(result.FailedSamples) != 1 {
		t.Fatalf("counts = success %d failure %d failed samples %d", result.SuccessCount, result.FailureCount, len(result.FailedSamples))
	}
	if result.Ordering.ReadyWaitedForFirstBrowserExitCount != 1 || result.Ordering.ReadyBeforeFirstBrowserExitCount != 1 || result.Ordering.ViolationSampleIndexes[0] != 1 {
		t.Fatalf("ordering = %#v", result.Ordering)
	}
	metric := result.Metrics["immediateReadyMs"]
	if metric.MinimumMs != 700 || metric.P50Ms != 700 || metric.P95Ms != 800 || metric.MaximumMs != 800 {
		t.Fatalf("immediateReadyMs = %#v", metric)
	}
	if result.Correlation.PearsonCoefficient == nil || *result.Correlation.PearsonCoefficient < 0.99 {
		t.Fatalf("correlation = %#v", result.Correlation)
	}
	if result.Environment.GitHubRunID == nil || *result.Environment.GitHubRunID != runID || result.Environment.GitHubRunAttempt == nil || *result.Environment.GitHubRunAttempt != runAttempt {
		t.Fatalf("environment = %#v", result.Environment)
	}
}

func TestSummarizeRejectsIncompleteSuccess(t *testing.T) {
	_, err := summarize(evidence{SchemaVersion: "velox.startup-lifecycle/v2", Scope: "fresh-and-immediate-same-profile-startup", Outcome: "success", Repetitions: 1, Samples: []sample{{Index: 0, Outcome: "success"}}}, nil)
	if err == nil {
		t.Fatal("summarize accepted an incomplete success sample")
	}
}

func TestSummarizeRejectsOutcomeMismatch(t *testing.T) {
	_, err := summarize(evidence{SchemaVersion: "velox.startup-lifecycle/v2", Scope: "fresh-and-immediate-same-profile-startup", Outcome: "success", Repetitions: 1, Samples: []sample{{Index: 0, Outcome: "failure", Error: &runError{Phase: "first-launch", Code: "HOST_RUN_FAILED"}}}}, nil)
	if err == nil {
		t.Fatal("summarize accepted an outcome that disagrees with its samples")
	}
}

func TestRunReadsCompleteLifecycleEvidence(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.json")
	output := filepath.Join(directory, "output.json")
	body := `{
  "schemaVersion":"velox.startup-lifecycle/v2",
  "scope":"fresh-and-immediate-same-profile-startup",
  "evidenceLevel":"hosted-runner-evidence",
  "outcome":"success",
  "repetitions":1,
  "startedAtUtc":"2026-07-13T00:00:00Z",
  "finishedAtUtc":"2026-07-13T00:00:01Z",
  "environment":{"os":"windows","architecture":"amd64","webView2Version":"1.2.3","runnerImage":"win25","runnerImageVersion":"1","githubRunId":"123","githubRunAttempt":"1","gitCommit":"1111111111111111111111111111111111111111"},
  "measurement":{"tool":"tests/startup/TestStartupLifecycleEvidence","toolVersion":1},
  "samples":[{"index":0,"outcome":"success","first":{"readyMs":100,"hostExitMs":10,"browserProcessId":1,"browserExitAfterHostMs":500},"immediate":{"readyMs":700,"hostExitMs":10,"browserProcessId":2,"browserExitAfterHostMs":500},"profileReleaseMs":500,"timeline":{"immediateProcessStartAfterFirstHostExitMs":2,"firstBrowserExitAfterImmediateStartMs":500,"immediateReadyAfterFirstBrowserExitMs":200,"immediateReadyWaitedForFirstBrowserExit":true},"error":null}]
}`
	if err := os.WriteFile(input, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--input", input, "--output", output}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsTrailingJSON(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.json")
	if err := os.WriteFile(input, []byte(`{"schemaVersion":"velox.startup-lifecycle/v2"}{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--input", input, "--output", filepath.Join(directory, "output.json")}); err == nil {
		t.Fatal("run accepted a trailing JSON value")
	}
}
