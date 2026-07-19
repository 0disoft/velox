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
		SchemaVersion: "velox.startup-lifecycle/v3",
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
	_, err := summarize(evidence{SchemaVersion: "velox.startup-lifecycle/v3", Scope: "fresh-and-immediate-same-profile-startup", Outcome: "success", Repetitions: 1, Samples: []sample{{Index: 0, Outcome: "success"}}}, nil)
	if err == nil {
		t.Fatal("summarize accepted an incomplete success sample")
	}
}

func TestSummarizeRejectsOutcomeMismatch(t *testing.T) {
	_, err := summarize(evidence{SchemaVersion: "velox.startup-lifecycle/v3", Scope: "fresh-and-immediate-same-profile-startup", Outcome: "success", Repetitions: 1, Samples: []sample{{Index: 0, Outcome: "failure", Error: &runError{Phase: "first-launch", Code: "HOST_RUN_FAILED"}}}}, nil)
	if err == nil {
		t.Fatal("summarize accepted an outcome that disagrees with its samples")
	}
}

func TestSummarizePhasesFindsImmediateStartupDominantInterval(t *testing.T) {
	startup := func(controller float64) phaseTimeline {
		elapsed := []float64{0, 1, 2, 3, 4, 5, controller, controller + 1, controller + 2, controller + 3, controller + 20}
		phases := make([]phasePoint, len(startupPhaseNames))
		for index, name := range startupPhaseNames {
			phases[index] = phasePoint{Name: name, ElapsedMS: elapsed[index]}
		}
		return phaseTimeline{SchemaVersion: "velox.host-startup-timeline/v1", Clock: "time-since-host-entry-monotonic", Phases: phases}
	}
	shutdownPhases := make([]phasePoint, len(shutdownPhaseNames))
	for index, name := range shutdownPhaseNames {
		shutdownPhases[index] = phasePoint{Name: name, ElapsedMS: float64(index)}
	}
	shutdown := phaseTimeline{SchemaVersion: "velox.host-shutdown-timeline/v1", Clock: "time-since-shutdown-request-monotonic", Phases: shutdownPhases}
	raw := evidence{
		SchemaVersion: "velox.startup-lifecycle/v3", EvidenceLevel: "hosted-runner-evidence", Outcome: "success",
		Samples: []sample{
			{Index: 0, Outcome: "success", First: &launch{StartupTimeline: startup(50), ShutdownTimeline: shutdown}, Immediate: &launch{StartupTimeline: startup(5800), ShutdownTimeline: shutdown}},
			{Index: 1, Outcome: "success", First: &launch{StartupTimeline: startup(60), ShutdownTimeline: shutdown}, Immediate: &launch{StartupTimeline: startup(5900), ShutdownTimeline: shutdown}},
		},
	}

	result, err := summarizePhases(raw, []byte("source"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Attribution.ImmediateStartupDominantInterval != "environment-created->controller-created" || result.Attribution.DominantSampleCount != 2 {
		t.Fatalf("attribution = %#v", result.Attribution)
	}
	metric := result.Groups["immediateStartup"].Intervals["environment-created->controller-created"]
	if metric.P50Ms != 5795 || metric.P95Ms != 5895 {
		t.Fatalf("controller interval = %#v", metric)
	}
}

func TestSummarizePhasesRejectsReorderedTimeline(t *testing.T) {
	startupPhases := make([]phasePoint, len(startupPhaseNames))
	for index, name := range startupPhaseNames {
		startupPhases[index] = phasePoint{Name: name, ElapsedMS: float64(index)}
	}
	startupPhases[6].ElapsedMS = -1
	startup := phaseTimeline{SchemaVersion: "velox.host-startup-timeline/v1", Clock: "time-since-host-entry-monotonic", Phases: startupPhases}
	shutdownPhases := make([]phasePoint, len(shutdownPhaseNames))
	for index, name := range shutdownPhaseNames {
		shutdownPhases[index] = phasePoint{Name: name, ElapsedMS: float64(index)}
	}
	shutdown := phaseTimeline{SchemaVersion: "velox.host-shutdown-timeline/v1", Clock: "time-since-shutdown-request-monotonic", Phases: shutdownPhases}
	_, err := summarizePhases(evidence{SchemaVersion: "velox.startup-lifecycle/v3", Samples: []sample{{Outcome: "success", First: &launch{StartupTimeline: startup, ShutdownTimeline: shutdown}, Immediate: &launch{StartupTimeline: startup, ShutdownTimeline: shutdown}}}}, nil)
	if err == nil {
		t.Fatal("summarizePhases accepted a reordered timeline")
	}
}

func TestRunReadsCompleteLifecycleEvidence(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.json")
	output := filepath.Join(directory, "output.json")
	body := `{
  "schemaVersion":"velox.startup-lifecycle/v3",
  "scope":"fresh-and-immediate-same-profile-startup",
  "evidenceLevel":"hosted-runner-evidence",
  "outcome":"success",
  "repetitions":1,
  "startedAtUtc":"2026-07-13T00:00:00Z",
  "finishedAtUtc":"2026-07-13T00:00:01Z",
  "environment":{"os":"windows","architecture":"amd64","webView2Version":"1.2.3","runnerImage":"win25","runnerImageVersion":"1","githubRunId":"123","githubRunAttempt":"1","gitCommit":"1111111111111111111111111111111111111111"},
  "measurement":{"tool":"tests/startup/TestStartupLifecycleEvidence","toolVersion":2},
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
	if err := os.WriteFile(input, []byte(`{"schemaVersion":"velox.startup-lifecycle/v3"}{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--input", input, "--output", filepath.Join(directory, "output.json")}); err == nil {
		t.Fatal("run accepted a trailing JSON value")
	}
}
