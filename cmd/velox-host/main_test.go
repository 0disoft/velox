package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/webview2"
)

func TestDefaultConfigPathIsExecutableScoped(t *testing.T) {
	executablePath := filepath.Join(t.TempDir(), "portable", "deskboard.exe")
	path, err := defaultConfigPath(func() (string, error) { return executablePath, nil })
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(executablePath), "velox.runtime.json")
	if path != want {
		t.Fatalf("defaultConfigPath() = %q, want %q", path, want)
	}
}

func TestDefaultConfigPathRejectsUnavailableOrRelativeExecutable(t *testing.T) {
	if _, err := defaultConfigPath(func() (string, error) { return "", errors.New("unavailable") }); err == nil {
		t.Fatal("defaultConfigPath accepted an unavailable executable path")
	}
	if _, err := defaultConfigPath(func() (string, error) { return "deskboard.exe", nil }); err == nil {
		t.Fatal("defaultConfigPath accepted a relative executable path")
	}
}

func TestBenchmarkOptionsRequireExplicitMode(t *testing.T) {
	values := map[string]string{
		"VELOX_BENCH_PIPE":                 `\\.\pipe\velox-test`,
		"VELOX_BENCH_POLICY_AUDIT":         "1",
		"VELOX_BENCH_EXIT_AFTER_READY":     "1",
		"VELOX_BENCH_WEBVIEW2_BROWSER_DIR": `C:\runtime`,
	}
	getenv := func(key string) string { return values[key] }

	disabled := benchmarkOptionsFromEnvironment(getenv)
	if disabled.enabled || disabled.pipeConfigured || disabled.policyAudit || disabled.exitAfterReady || disabled.browserExecutableFolder != "" {
		t.Fatalf("benchmark options leaked without explicit mode: %+v", disabled)
	}

	values["VELOX_BENCH_MODE"] = "1"
	enabled := benchmarkOptionsFromEnvironment(getenv)
	if !enabled.enabled || !enabled.pipeConfigured || !enabled.policyAudit || !enabled.exitAfterReady || enabled.browserExecutableFolder != `C:\runtime` {
		t.Fatalf("benchmark options not loaded in explicit mode: %+v", enabled)
	}
}

func TestDefaultDataPathIsStableAndAppScoped(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LocalAppData", base)
	path, err := defaultDataPath("dev.velox.hello")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "Velox", "profiles", "dev.velox.hello")
	if path != want || !filepath.IsAbs(path) {
		t.Fatalf("defaultDataPath() = %q, want %q", path, want)
	}
}

func TestPolicyAuditRequiresIPCAndEveryBrowserPolicy(t *testing.T) {
	audit := newPolicyAudit(true)
	completed := 0
	audit.complete = func() { completed++ }
	for _, policy := range []string{
		webview2.PolicyNavigation,
		webview2.PolicyFrameNavigation,
		webview2.PolicyNewWindow,
		webview2.PolicyDownload,
		webview2.PolicyPermission,
	} {
		audit.record(policy)
	}
	if completed != 0 {
		t.Fatal("policy audit completed before the trusted IPC fixture passed")
	}
	audit.markIPCReady("ipc-ok")
	audit.markIPCReady("ipc-ok")
	if completed != 1 || len(audit.missing()) != 0 {
		t.Fatalf("policy audit completed=%d missing=%v", completed, audit.missing())
	}
}
