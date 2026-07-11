package runner

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

func TestExecuteProvidesValidTemporaryConfigAndRemovesIt(t *testing.T) {
	plan := runnerPlan(t)
	var observedPath string
	result, err := Execute(plan, func(hostPath, configPath string, stdout, stderr io.Writer) (int, error) {
		observedPath = configPath
		if filepath.Dir(configPath) != plan.Snapshot().Manifest.ProjectRoot {
			t.Fatalf("config outside project root: %s", configPath)
		}
		if _, err := os.Stat(configPath); err != nil {
			t.Fatalf("temporary config unavailable to host: %v", err)
		}
		resolved, err := runtimeconfig.Load(configPath)
		if err != nil {
			t.Fatalf("temporary config is not host-readable: %v", err)
		}
		if resolved.AssetRoot != plan.Snapshot().Manifest.AssetRoot {
			t.Fatalf("asset root = %q, want %q", resolved.AssetRoot, plan.Snapshot().Manifest.AssetRoot)
		}
		return 0, nil
	}, io.Discard, io.Discard)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(observedPath); !os.IsNotExist(err) {
		t.Fatalf("temporary config remained: %v", err)
	}
}

func TestExecutePreservesHostExitCodeAndCleansConfig(t *testing.T) {
	plan := runnerPlan(t)
	var observedPath string
	result, err := Execute(plan, func(hostPath, configPath string, stdout, stderr io.Writer) (int, error) {
		observedPath = configPath
		return 5, nil
	}, io.Discard, io.Discard)
	var exitError *HostExitError
	if result.ExitCode != 5 || !errors.As(err, &exitError) || exitError.Code != 5 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(observedPath); !os.IsNotExist(err) {
		t.Fatalf("temporary config remained: %v", err)
	}
}

func TestExecuteCleansConfigWhenHostCannotStart(t *testing.T) {
	plan := runnerPlan(t)
	var observedPath string
	result, err := Execute(plan, func(hostPath, configPath string, stdout, stderr io.Writer) (int, error) {
		observedPath = configPath
		return 6, errors.New("start failed")
	}, io.Discard, io.Discard)
	if result.ExitCode != 6 || err == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(observedPath); !os.IsNotExist(err) {
		t.Fatalf("temporary config remained: %v", err)
	}
}

func runnerPlan(t *testing.T) buildplan.Plan {
	t.Helper()
	root := t.TempDir()
	host := filepath.Join(root, "release", "velox-host.exe")
	if err := os.MkdirAll(filepath.Dir(host), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(host, []byte("host"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("host"))
	metadata := fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.3.0-dev","target":"windows-x64","contracts":{"host":1,"runtime":1},"host":{"file":"velox-host.exe","bytes":4,"sha256":"%x"}}`, digest)
	if err := os.WriteFile(filepath.Join(filepath.Dir(host), "velox-host.json"), []byte(metadata), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "site"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site", "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "velox.json")
	if err := os.WriteFile(manifestPath, []byte(`{"schemaVersion":1,"app":{"id":"com.example.run","name":"Run","version":"1.0.0"},"assets":{"root":"site","entry":"index.html"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: host, Target: buildplan.TargetWindowsX64, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}
