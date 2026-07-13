package startup_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	veloxwebview2 "github.com/0disoft/velox/internal/webview2"
)

const (
	lifecycleSchemaVersion = "velox.startup-lifecycle/v2"
	lifecycleResultEnv     = "VELOX_STARTUP_LIFECYCLE_RESULT"
	lifecycleRepetitionEnv = "VELOX_STARTUP_LIFECYCLE_REPETITIONS"
)

type lifecycleEvidence struct {
	SchemaVersion string               `json:"schemaVersion"`
	Scope         string               `json:"scope"`
	EvidenceLevel string               `json:"evidenceLevel"`
	Outcome       string               `json:"outcome"`
	Repetitions   int                  `json:"repetitions"`
	StartedAtUTC  time.Time            `json:"startedAtUtc"`
	FinishedAtUTC time.Time            `json:"finishedAtUtc"`
	Environment   lifecycleEnvironment `json:"environment"`
	Measurement   lifecycleMeasurement `json:"measurement"`
	Samples       []lifecycleSample    `json:"samples"`
}

type lifecycleEnvironment struct {
	OS                 string  `json:"os"`
	Architecture       string  `json:"architecture"`
	WebView2Version    string  `json:"webView2Version"`
	RunnerImage        *string `json:"runnerImage"`
	RunnerImageVersion *string `json:"runnerImageVersion"`
	GitHubRunID        *string `json:"githubRunId"`
	GitHubRunAttempt   *string `json:"githubRunAttempt"`
	GitCommit          *string `json:"gitCommit"`
}

type lifecycleMeasurement struct {
	Tool                    string `json:"tool"`
	ToolVersion             int    `json:"toolVersion"`
	Unit                    string `json:"unit"`
	Clock                   string `json:"clock"`
	Concurrency             int    `json:"concurrency"`
	FreshProfilePerSample   bool   `json:"freshProfilePerSample"`
	ImmediateRelaunch       bool   `json:"immediateRelaunch"`
	ReadyBoundary           string `json:"readyBoundary"`
	BrowserExitBoundary     string `json:"browserExitBoundary"`
	ProfileReleaseBoundary  string `json:"profileReleaseBoundary"`
	HostExitTimeoutMs       int64  `json:"hostExitTimeoutMs"`
	BrowserExitTimeoutMs    int64  `json:"browserExitTimeoutMs"`
	ProfileReleaseTimeoutMs int64  `json:"profileReleaseTimeoutMs"`
}

type lifecycleSample struct {
	Index            int                `json:"index"`
	Outcome          string             `json:"outcome"`
	First            *lifecycleLaunch   `json:"first"`
	Immediate        *lifecycleLaunch   `json:"immediate"`
	ProfileReleaseMs *float64           `json:"profileReleaseMs"`
	Timeline         *lifecycleTimeline `json:"timeline"`
	Error            *lifecycleError    `json:"error"`
}

type lifecycleTimeline struct {
	ImmediateProcessStartAfterFirstHostExitMs float64 `json:"immediateProcessStartAfterFirstHostExitMs"`
	FirstBrowserExitAfterImmediateStartMs     float64 `json:"firstBrowserExitAfterImmediateStartMs"`
	ImmediateReadyAfterFirstBrowserExitMs     float64 `json:"immediateReadyAfterFirstBrowserExitMs"`
	ImmediateReadyWaitedForFirstBrowserExit   bool    `json:"immediateReadyWaitedForFirstBrowserExit"`
}

type lifecycleLaunch struct {
	ReadyMs                float64 `json:"readyMs"`
	HostExitMs             float64 `json:"hostExitMs"`
	BrowserProcessID       uint32  `json:"browserProcessId"`
	BrowserExitAfterHostMs float64 `json:"browserExitAfterHostMs"`
}

type lifecycleError struct {
	Phase string `json:"phase"`
	Code  string `json:"code"`
}

func TestStartupLifecycleEvidence(t *testing.T) {
	resultPath := os.Getenv(lifecycleResultEnv)
	if resultPath == "" {
		t.Skip(lifecycleResultEnv + " is set only by the hosted lifecycle evidence workflow")
	}

	repetitions := lifecycleRepetitions(t)
	repoRoot := repositoryRoot(t)
	host := goHost(t, repoRoot)
	webView2Version, err := veloxwebview2.InstalledVersion()
	if err != nil {
		t.Fatalf("read WebView2 version: %v", err)
	}

	evidence := lifecycleEvidence{
		SchemaVersion: lifecycleSchemaVersion,
		Scope:         "fresh-and-immediate-same-profile-startup",
		EvidenceLevel: lifecycleEvidenceLevel(),
		Outcome:       "success",
		Repetitions:   repetitions,
		StartedAtUTC:  time.Now().UTC(),
		Environment: lifecycleEnvironment{
			OS: runtime.GOOS, Architecture: runtime.GOARCH, WebView2Version: webView2Version,
			RunnerImage: optionalEnvironment("ImageOS"), RunnerImageVersion: optionalEnvironment("ImageVersion"),
			GitHubRunID: optionalEnvironment("GITHUB_RUN_ID"), GitHubRunAttempt: optionalEnvironment("GITHUB_RUN_ATTEMPT"),
			GitCommit: optionalEnvironment("GITHUB_SHA"),
		},
		Measurement: lifecycleMeasurement{
			Tool: "tests/startup/TestStartupLifecycleEvidence", ToolVersion: 1,
			Unit: "milliseconds", Clock: "time.Time-with-process-local-monotonic-component", Concurrency: 1,
			FreshProfilePerSample: true, ImmediateRelaunch: true,
			ReadyBoundary:          "process-start-to-domcontentloaded-plus-two-animation-frames",
			BrowserExitBoundary:    "host-exit-to-signaled-main-browser-process-handle",
			ProfileReleaseBoundary: "immediate-host-exit-to-successful-user-data-folder-removal",
			HostExitTimeoutMs:      5000, BrowserExitTimeoutMs: 10000, ProfileReleaseTimeoutMs: 10000,
		},
		Samples: make([]lifecycleSample, 0, repetitions),
	}

	failures := 0
	for index := 0; index < repetitions; index++ {
		sample := measureLifecycleSample(repoRoot, host, index)
		if sample.Outcome != "success" {
			failures++
			evidence.Outcome = "failure"
		}
		evidence.Samples = append(evidence.Samples, sample)
	}
	evidence.FinishedAtUTC = time.Now().UTC()

	if err := writeLifecycleEvidence(resultPath, evidence); err != nil {
		t.Fatalf("write startup lifecycle evidence: %v", err)
	}
	if failures > 0 {
		t.Fatalf("%d of %d startup lifecycle samples failed; evidence preserved at %s", failures, repetitions, resultPath)
	}
}

func measureLifecycleSample(repoRoot string, host hostAdapter, index int) lifecycleSample {
	sample := lifecycleSample{Index: index, Outcome: "failure"}
	profileBase := filepath.Join(repoRoot, ".cache", "profiles")
	if err := os.MkdirAll(profileBase, 0o755); err != nil {
		return failLifecycleSample(sample, "profile-create", "PROFILE_CREATE_FAILED")
	}
	profile, err := os.MkdirTemp(profileBase, fmt.Sprintf("velox-lifecycle-%02d-", index))
	if err != nil {
		return failLifecycleSample(sample, "profile-create", "PROFILE_CREATE_FAILED")
	}

	first, err := runHost(host, profile)
	if err != nil {
		_, _ = waitForProfileRelease(profile, 10*time.Second)
		return failLifecycleSample(sample, "first-launch", "HOST_RUN_FAILED")
	}
	sample.First = launchWithoutBrowserExit(first)

	immediate, err := runHost(host, profile)
	if err != nil {
		_, _ = awaitBrowserExit(first, 10*time.Second)
		_, _ = waitForProfileRelease(profile, 10*time.Second)
		return failLifecycleSample(sample, "immediate-launch", "HOST_RUN_FAILED")
	}
	sample.Immediate = launchWithoutBrowserExit(immediate)

	profileReleaseStarted := time.Now()
	profileRelease, profileErr := waitForProfileRelease(profile, 10*time.Second)
	firstBrowserExitedAt, firstErr := awaitBrowserExitAt(first, 10*time.Second)
	immediateBrowserExitedAt, immediateErr := awaitBrowserExitAt(immediate, 10*time.Second)
	if firstErr != nil {
		return failLifecycleSample(sample, "first-browser-exit", "BROWSER_EXIT_FAILED")
	}
	sample.First.BrowserExitAfterHostMs = milliseconds(firstBrowserExitedAt.Sub(first.HostExitedAt))
	if immediateErr != nil {
		return failLifecycleSample(sample, "immediate-browser-exit", "BROWSER_EXIT_FAILED")
	}
	sample.Immediate.BrowserExitAfterHostMs = milliseconds(immediateBrowserExitedAt.Sub(immediate.HostExitedAt))
	if profileErr != nil {
		return failLifecycleSample(sample, "profile-release", "PROFILE_RELEASE_FAILED")
	}
	profileReleasedAfterHost := profileReleaseStarted.Add(profileRelease).Sub(immediate.HostExitedAt)
	value := milliseconds(profileReleasedAfterHost)
	sample.ProfileReleaseMs = &value
	sample.Timeline = &lifecycleTimeline{
		ImmediateProcessStartAfterFirstHostExitMs: milliseconds(immediate.ProcessStartedAt.Sub(first.HostExitedAt)),
		FirstBrowserExitAfterImmediateStartMs:     milliseconds(firstBrowserExitedAt.Sub(immediate.ProcessStartedAt)),
		ImmediateReadyAfterFirstBrowserExitMs:     milliseconds(immediate.ReadyAt.Sub(firstBrowserExitedAt)),
		ImmediateReadyWaitedForFirstBrowserExit:   !immediate.ReadyAt.Before(firstBrowserExitedAt),
	}
	sample.Outcome = "success"
	return sample
}

func launchWithoutBrowserExit(run hostRun) *lifecycleLaunch {
	return &lifecycleLaunch{
		ReadyMs: milliseconds(run.Ready), HostExitMs: milliseconds(run.Exit),
		BrowserProcessID: run.BrowserProcessID,
	}
}

func failLifecycleSample(sample lifecycleSample, phase, code string) lifecycleSample {
	sample.Error = &lifecycleError{Phase: phase, Code: code}
	return sample
}

func lifecycleRepetitions(t *testing.T) int {
	t.Helper()
	value := os.Getenv(lifecycleRepetitionEnv)
	if value == "" {
		return 10
	}
	repetitions, err := strconv.Atoi(value)
	if err != nil || repetitions < 1 || repetitions > 100 {
		t.Fatalf("%s must be an integer from 1 through 100", lifecycleRepetitionEnv)
	}
	return repetitions
}

func lifecycleEvidenceLevel() string {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return "hosted-runner-evidence"
	}
	return "controlled-local-observation"
}

func optionalEnvironment(name string) *string {
	value := os.Getenv(name)
	if value == "" {
		return nil
	}
	return &value
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func writeLifecycleEvidence(path string, evidence lifecycleEvidence) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, body, 0o644); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}
