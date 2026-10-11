package startup_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/velox/internal/inspector"
	"github.com/0disoft/velox/internal/runtimeconfig"
	veloxwebview2 "github.com/0disoft/velox/internal/webview2"
)

func TestPackagedAppLifecycle(t *testing.T) {
	directory := os.Getenv("VELOX_PACKAGED_APP_DIRECTORY")
	if directory == "" {
		t.Skip("VELOX_PACKAGED_APP_DIRECTORY enables this optional packaged-app check")
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := inspector.Inspect(directory)
	if err != nil {
		t.Fatalf("inspect packaged input: %v", err)
	}
	configPath := filepath.Join(directory, "velox.runtime.json")
	config, err := runtimeconfig.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.App.ID != artifact.App.ID {
		t.Fatal("packaged runtime identity differs from inspected artifact")
	}
	host := hostAdapter{
		name: "packaged-app", executable: filepath.Join(directory, artifact.App.ID+".exe"),
		arguments: func(string) []string { return []string{"--config", configPath} },
		environment: func(profile string) []string {
			return []string{"VELOX_DATA_DIR=" + profile}
		},
		expectedPhase: "dom-2raf", requireBrowserProcess: true,
	}
	version, err := veloxwebview2.InstalledVersion()
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC()
	sample := packagedLifecycleGuardrails(measureLifecycleSample(repositoryRoot(t), host, 0, t.Logf))
	if output := os.Getenv("VELOX_PACKAGED_APP_LIFECYCLE_RESULT"); output != "" {
		receipt := struct {
			SchemaVersion   string           `json:"schemaVersion"`
			StartedAtUTC    time.Time        `json:"startedAtUtc"`
			FinishedAtUTC   time.Time        `json:"finishedAtUtc"`
			Artifact        inspector.Result `json:"artifact"`
			WebView2Version string           `json:"webView2Version"`
			Sample          lifecycleSample  `json:"sample"`
		}{"velox.packaged-app-lifecycle/v1", started, time.Now().UTC(), artifact, version, sample}
		data, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(output, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if sample.Outcome != "success" {
		t.Fatalf("packaged app lifecycle failed: %+v", sample.Error)
	}
	t.Logf("app=%s host=%s assets=%s first-ready=%.2fms immediate-ready=%.2fms immediate-host-exit=%.2fms browser-exit-after-host=%.2fms profile-release=%.2fms",
		artifact.App.ID, artifact.HostSHA256, artifact.AssetSHA256, sample.First.ReadyMs,
		sample.Immediate.ReadyMs, sample.Immediate.HostExitMs, sample.Immediate.BrowserExitAfterHostMs, *sample.ProfileReleaseMs)
}

func packagedLifecycleGuardrails(sample lifecycleSample) lifecycleSample {
	if sample.Outcome != "success" {
		return sample
	}
	var phase, code string
	switch {
	case sample.First == nil || sample.Immediate == nil || sample.ProfileReleaseMs == nil:
		phase, code = "evidence", "LIFECYCLE_EVIDENCE_INVALID"
	case sample.First.HostExitMs > 1000:
		phase, code = "first-host-exit", "HOST_EXIT_FAILED"
	case sample.Immediate.HostExitMs > 1000:
		phase, code = "immediate-host-exit", "HOST_EXIT_FAILED"
	case sample.Immediate.ReadyMs > 10000:
		phase, code = "immediate-launch", "HOST_RUN_FAILED"
	default:
		return sample
	}
	sample.Outcome = "failure"
	sample.Error = &lifecycleError{Phase: phase, Code: code}
	return sample
}

func TestPackagedLifecycleGuardrails(t *testing.T) {
	for _, test := range []struct {
		name, phase string
		change      func(*lifecycleSample)
	}{
		{"first-host-exit", "first-host-exit", func(s *lifecycleSample) { s.First.HostExitMs = 1001 }},
		{"immediate-host-exit", "immediate-host-exit", func(s *lifecycleSample) { s.Immediate.HostExitMs = 1001 }},
		{"immediate-ready", "immediate-launch", func(s *lifecycleSample) { s.Immediate.ReadyMs = 10001 }},
		{"missing-launch", "evidence", func(s *lifecycleSample) { s.First = nil }},
		{"missing-release", "evidence", func(s *lifecycleSample) { s.ProfileReleaseMs = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			released := float64(10)
			sample := lifecycleSample{Outcome: "success", First: &lifecycleLaunch{}, Immediate: &lifecycleLaunch{}, ProfileReleaseMs: &released}
			if got := packagedLifecycleGuardrails(sample); got.Outcome != "success" {
				t.Fatal("valid sample was rejected")
			}
			test.change(&sample)
			got := packagedLifecycleGuardrails(sample)
			if got.Outcome != "failure" || got.Error == nil || got.Error.Phase != test.phase {
				t.Fatalf("failed guardrail would produce a success receipt: %+v", got)
			}
		})
	}
	failed := lifecycleSample{Outcome: "failure", Error: &lifecycleError{Phase: "first-browser-exit", Code: "BROWSER_EXIT_FAILED"}}
	if got := packagedLifecycleGuardrails(failed); got.Error != failed.Error || got.Outcome != "failure" {
		t.Fatal("original failure diagnostic was lost")
	}
}
