package startup_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"time"

	actutumwebview2 "github.com/0disoft/actutum/internal/webview2"
)

const (
	profileComparisonSchemaVersion = "actutum.startup-profile-comparison/v1"
	profileComparisonResultEnv     = "ACTUTUM_STARTUP_PROFILE_COMPARISON_RESULT"
	profileComparisonRepetitionEnv = "ACTUTUM_STARTUP_PROFILE_COMPARISON_REPETITIONS"
)

type profileComparisonEvidence struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Scope         string                    `json:"scope"`
	EvidenceLevel string                    `json:"evidenceLevel"`
	Outcome       string                    `json:"outcome"`
	Repetitions   int                       `json:"repetitions"`
	StartedAtUTC  time.Time                 `json:"startedAtUtc"`
	FinishedAtUTC time.Time                 `json:"finishedAtUtc"`
	Environment   lifecycleEnvironment      `json:"environment"`
	Measurement   profileComparisonContract `json:"measurement"`
	Samples       []profileComparisonSample `json:"samples"`
	Summary       *profileComparisonSummary `json:"summary"`
}

type profileComparisonContract struct {
	Tool                  string `json:"tool"`
	ToolVersion           int    `json:"toolVersion"`
	Unit                  string `json:"unit"`
	ReadyBoundary         string `json:"readyBoundary"`
	AlternatingTrialOrder bool   `json:"alternatingTrialOrder"`
	ConcurrentTrials      bool   `json:"concurrentTrials"`
}

type profileComparisonSample struct {
	Index        int                     `json:"index"`
	Outcome      string                  `json:"outcome"`
	Order        []string                `json:"order"`
	SameProfile  *profileComparisonTrial `json:"sameProfile"`
	FreshProfile *profileComparisonTrial `json:"freshProfile"`
	Error        *lifecycleError         `json:"error"`
}

type profileComparisonTrial struct {
	Mode                                   string  `json:"mode"`
	FirstReadyMs                           float64 `json:"firstReadyMs"`
	SecondReadyMs                          float64 `json:"secondReadyMs"`
	FirstBrowserExitAfterSecondStartMs     float64 `json:"firstBrowserExitAfterSecondStartMs"`
	SecondReadyAfterFirstBrowserExitMs     float64 `json:"secondReadyAfterFirstBrowserExitMs"`
	SecondReadyWaitedForFirstBrowserExit   bool    `json:"secondReadyWaitedForFirstBrowserExit"`
	SecondBrowserExitAfterSecondHostExitMs float64 `json:"secondBrowserExitAfterSecondHostExitMs"`
}

type profileComparisonSummary struct {
	SuccessfulPairs         int     `json:"successfulPairs"`
	SameProfileSecondP50Ms  float64 `json:"sameProfileSecondP50Ms"`
	FreshProfileSecondP50Ms float64 `json:"freshProfileSecondP50Ms"`
	P50DeltaMs              float64 `json:"p50DeltaMs"`
}

func TestStartupProfileComparisonEvidence(t *testing.T) {
	resultPath := os.Getenv(profileComparisonResultEnv)
	if resultPath == "" {
		t.Skip(profileComparisonResultEnv + " is set only by an explicit profile-comparison workflow")
	}
	repetitions := profileComparisonRepetitions(t)
	repoRoot := repositoryRoot(t)
	host := goHost(t, repoRoot)
	webView2Version, err := actutumwebview2.InstalledVersion()
	if err != nil {
		t.Fatalf("read WebView2 version: %v", err)
	}
	evidence := profileComparisonEvidence{
		SchemaVersion: profileComparisonSchemaVersion,
		Scope:         "same-profile-versus-fresh-profile-immediate-relaunch",
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
		Measurement: profileComparisonContract{
			Tool: "tests/startup/TestStartupProfileComparisonEvidence", ToolVersion: 1,
			Unit: "milliseconds", ReadyBoundary: "process-start-to-domcontentloaded-plus-two-animation-frames",
			AlternatingTrialOrder: true, ConcurrentTrials: false,
		},
		Samples: make([]profileComparisonSample, 0, repetitions),
	}

	var sameReady, freshReady []float64
	for index := 0; index < repetitions; index++ {
		sample := measureProfileComparisonSample(repoRoot, host, index)
		if sample.Outcome == "success" {
			sameReady = append(sameReady, sample.SameProfile.SecondReadyMs)
			freshReady = append(freshReady, sample.FreshProfile.SecondReadyMs)
		} else {
			evidence.Outcome = "failure"
		}
		evidence.Samples = append(evidence.Samples, sample)
	}
	if len(sameReady) > 0 {
		sameP50, freshP50 := profileComparisonP50(sameReady), profileComparisonP50(freshReady)
		evidence.Summary = &profileComparisonSummary{
			SuccessfulPairs: len(sameReady), SameProfileSecondP50Ms: sameP50,
			FreshProfileSecondP50Ms: freshP50, P50DeltaMs: sameP50 - freshP50,
		}
	}
	evidence.FinishedAtUTC = time.Now().UTC()
	if err := writeProfileComparisonEvidence(resultPath, evidence); err != nil {
		t.Fatalf("write startup profile comparison: %v", err)
	}
	if evidence.Outcome != "success" {
		t.Fatalf("startup profile comparison failed; evidence preserved at %s", resultPath)
	}
}

func measureProfileComparisonSample(repoRoot string, host hostAdapter, index int) profileComparisonSample {
	order := []string{"same-profile", "fresh-profile"}
	if index%2 == 1 {
		order[0], order[1] = order[1], order[0]
	}
	sample := profileComparisonSample{Index: index, Outcome: "failure", Order: order}
	for _, mode := range order {
		trial, err := measureProfileComparisonTrial(repoRoot, host, index, mode)
		if err != nil {
			sample.Error = &lifecycleError{Phase: mode, Code: "TRIAL_FAILED"}
			return sample
		}
		if mode == "same-profile" {
			sample.SameProfile = trial
		} else {
			sample.FreshProfile = trial
		}
	}
	sample.Outcome = "success"
	return sample
}

func measureProfileComparisonTrial(repoRoot string, host hostAdapter, index int, mode string) (*profileComparisonTrial, error) {
	profileBase := filepath.Join(repoRoot, ".cache", "profiles")
	if err := os.MkdirAll(profileBase, 0o755); err != nil {
		return nil, err
	}
	firstProfile, err := os.MkdirTemp(profileBase, fmt.Sprintf("actutum-profile-comparison-%02d-first-", index))
	if err != nil {
		return nil, err
	}
	secondProfile := firstProfile
	if mode == "fresh-profile" {
		secondProfile, err = os.MkdirTemp(profileBase, fmt.Sprintf("actutum-profile-comparison-%02d-second-", index))
		if err != nil {
			_ = os.RemoveAll(firstProfile)
			return nil, err
		}
	}

	first, err := runHost(host, firstProfile)
	if err != nil {
		cleanupComparisonProfiles(firstProfile, secondProfile)
		return nil, err
	}
	second, err := runHost(host, secondProfile)
	if err != nil {
		_, _ = awaitBrowserExit(first, 10*time.Second)
		cleanupComparisonProfiles(firstProfile, secondProfile)
		return nil, err
	}
	firstBrowserExitedAt, firstErr := awaitBrowserExitAt(first, 10*time.Second)
	secondBrowserExitedAt, secondErr := awaitBrowserExitAt(second, 10*time.Second)
	if firstErr != nil || secondErr != nil {
		cleanupComparisonProfiles(firstProfile, secondProfile)
		return nil, fmt.Errorf("browser exit: first=%v second=%v", firstErr, secondErr)
	}
	if _, err := waitForProfileRelease(firstProfile, 10*time.Second); err != nil {
		cleanupComparisonProfiles(firstProfile, secondProfile)
		return nil, err
	}
	if secondProfile != firstProfile {
		if _, err := waitForProfileRelease(secondProfile, 10*time.Second); err != nil {
			cleanupComparisonProfiles(firstProfile, secondProfile)
			return nil, err
		}
	}
	return &profileComparisonTrial{
		Mode: mode, FirstReadyMs: milliseconds(first.Ready), SecondReadyMs: milliseconds(second.Ready),
		FirstBrowserExitAfterSecondStartMs:     milliseconds(firstBrowserExitedAt.Sub(second.ProcessStartedAt)),
		SecondReadyAfterFirstBrowserExitMs:     milliseconds(second.ReadyAt.Sub(firstBrowserExitedAt)),
		SecondReadyWaitedForFirstBrowserExit:   !second.ReadyAt.Before(firstBrowserExitedAt),
		SecondBrowserExitAfterSecondHostExitMs: milliseconds(secondBrowserExitedAt.Sub(second.HostExitedAt)),
	}, nil
}

func cleanupComparisonProfiles(paths ...string) {
	seen := make(map[string]struct{})
	for _, path := range paths {
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		_, _ = waitForProfileRelease(path, 10*time.Second)
	}
}

func profileComparisonRepetitions(t *testing.T) int {
	t.Helper()
	value := os.Getenv(profileComparisonRepetitionEnv)
	if value == "" {
		return 3
	}
	repetitions, err := strconv.Atoi(value)
	if err != nil || repetitions < 1 || repetitions > 10 {
		t.Fatalf("%s must be an integer from 1 through 10", profileComparisonRepetitionEnv)
	}
	return repetitions
}

func profileComparisonP50(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return sorted[(len(sorted)-1)/2]
}

func writeProfileComparisonEvidence(path string, evidence profileComparisonEvidence) error {
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
