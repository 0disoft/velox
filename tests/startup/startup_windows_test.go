package startup_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	pipeAccessInbound = 0x00000001
	pipeTypeByte      = 0x00000000
	pipeWait          = 0x00000000
	benchmarkRuns     = 10
	warmupRuns        = 5
)

var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	createNamedPipeW    = kernel32.NewProc("CreateNamedPipeW")
	connectNamedPipe    = kernel32.NewProc("ConnectNamedPipe")
	disconnectNamedPipe = kernel32.NewProc("DisconnectNamedPipe")
	cancelIoEx          = kernel32.NewProc("CancelIoEx")
)

type hostAdapter struct {
	name        string
	executable  string
	arguments   func(profile string) []string
	environment func(profile string) []string
	expected    string
}

type benchmarkResult struct {
	SchemaVersion int          `json:"schemaVersion"`
	RecordedAt    string       `json:"recordedAt"`
	Runs          int          `json:"runs"`
	Warmups       int          `json:"warmups"`
	Hosts         []hostResult `json:"hosts"`
}

type hostResult struct {
	Name            string        `json:"name"`
	ExecutableBytes int64         `json:"executableBytes"`
	Fresh           profileResult `json:"fresh"`
	Warm            profileResult `json:"warm"`
}

type profileResult struct {
	DurationsMS []float64 `json:"durationsMs"`
	P50MS       float64   `json:"p50Ms"`
	P95MS       float64   `json:"p95Ms"`
}

type hostRun struct {
	Ready time.Duration
	Exit  time.Duration
}

func TestBuiltHostStartup(t *testing.T) {
	repoRoot := repositoryRoot(t)
	host := goHost(t, repoRoot)
	profile := managedProfileRoot(t, "velox-go-smoke-")
	first := runHost(t, host, profile)
	immediate := runHost(t, host, profile)
	profileRelease := waitForProfileRelease(t, profile, 10*time.Second)
	securityProfile := managedProfileRoot(t, "velox-go-security-")
	security := runHost(t, securityHost(t, repoRoot), securityProfile)

	if first.Exit > time.Second || immediate.Exit > time.Second {
		t.Fatalf("host shutdown exceeded 1s: first=%s immediate=%s", first.Exit, immediate.Exit)
	}
	if immediate.Ready > 10*time.Second {
		t.Fatalf("same-profile immediate relaunch exceeded 10s: %s", immediate.Ready)
	}
	t.Logf("first ready=%s exit=%s; immediate ready=%s exit=%s; profile release=%s",
		first.Ready, first.Exit, immediate.Ready, immediate.Exit, profileRelease)
	t.Logf("security ready=%s exit=%s", security.Ready, security.Exit)
}

func TestBuiltCppHostStartup(t *testing.T) {
	repoRoot := repositoryRoot(t)
	host := cppHost(t, repoRoot)
	profile := managedProfileRoot(t, "velox-cpp-smoke-")
	run := runHost(t, host, profile)
	t.Logf("process-to-ready: %s; ready-to-exit: %s", run.Ready, run.Exit)
}

func TestHostStartupComparison(t *testing.T) {
	repoRoot := repositoryRoot(t)
	profileRoot := managedProfileRoot(t, "velox-startup-benchmark-")
	hosts := []hostAdapter{goHost(t, repoRoot), cppHost(t, repoRoot)}
	results := benchmarkResult{
		SchemaVersion: 1,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
		Runs:          benchmarkRuns,
		Warmups:       warmupRuns,
	}

	for _, host := range hosts {
		result := hostResult{Name: host.name}
		info, err := os.Stat(host.executable)
		if err != nil {
			t.Fatal(err)
		}
		result.ExecutableBytes = info.Size()

		freshDurations := make([]time.Duration, 0, benchmarkRuns)
		for run := 0; run < benchmarkRuns; run++ {
			profile := filepath.Join(profileRoot, host.name, fmt.Sprintf("fresh-%02d", run))
			if err := os.MkdirAll(profile, 0o755); err != nil {
				t.Fatal(err)
			}
			freshDurations = append(freshDurations, runHost(t, host, profile).Ready)
		}
		result.Fresh = summarize(freshDurations)

		warmProfile := filepath.Join(profileRoot, host.name, "warm")
		if err := os.MkdirAll(warmProfile, 0o755); err != nil {
			t.Fatal(err)
		}
		for run := 0; run < warmupRuns; run++ {
			runHost(t, host, warmProfile)
		}
		warmDurations := make([]time.Duration, 0, benchmarkRuns)
		for run := 0; run < benchmarkRuns; run++ {
			warmDurations = append(warmDurations, runHost(t, host, warmProfile).Ready)
		}
		result.Warm = summarize(warmDurations)
		results.Hosts = append(results.Hosts, result)
	}

	output := os.Getenv("VELOX_BENCH_OUTPUT")
	if output == "" {
		t.Skip("VELOX_BENCH_OUTPUT is set only by the startup benchmark intent")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(output, data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, result := range results.Hosts {
		t.Logf("%s fresh p50=%.2fms p95=%.2fms warm p50=%.2fms p95=%.2fms",
			result.Name, result.Fresh.P50MS, result.Fresh.P95MS, result.Warm.P50MS, result.Warm.P95MS)
	}
}

func managedProfileRoot(t *testing.T, pattern string) string {
	t.Helper()
	base := filepath.Join(repositoryRoot(t), ".cache", "profiles")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, pattern)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deadline := time.Now().Add(10 * time.Second)
		for {
			err := os.RemoveAll(root)
			if err == nil || os.IsNotExist(err) {
				return
			}
			if time.Now().After(deadline) {
				t.Logf("M0 WebView2 profile remained locked after host exit: %s: %v", root, err)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
	return root
}

func waitForProfileRelease(t *testing.T, root string, timeout time.Duration) time.Duration {
	t.Helper()
	started := time.Now()
	deadline := started.Add(timeout)
	for {
		err := os.RemoveAll(root)
		if err == nil || os.IsNotExist(err) {
			return time.Since(started)
		}
		if time.Now().After(deadline) {
			t.Fatalf("WebView2 profile remained locked after %s: %s: %v", timeout, root, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func goHost(t *testing.T, repoRoot string) hostAdapter {
	t.Helper()
	executable := requiredExecutable(t, "VELOX_BUILT_HOST")
	config := filepath.Join(repoRoot, "examples", "hello", "velox.runtime.json")
	return hostAdapter{
		name:       "go",
		executable: executable,
		arguments: func(string) []string {
			return []string{"--config", config}
		},
		environment: func(profile string) []string {
			return []string{"VELOX_DATA_DIR=" + profile}
		},
		expected: "ready dom-2raf\n",
	}
}

func securityHost(t *testing.T, repoRoot string) hostAdapter {
	t.Helper()
	executable := requiredExecutable(t, "VELOX_BUILT_HOST")
	config := filepath.Join(repoRoot, "tests", "fixtures", "security", "velox.runtime.json")
	return hostAdapter{
		name:       "go-security",
		executable: executable,
		arguments: func(string) []string {
			return []string{"--config", config}
		},
		environment: func(profile string) []string {
			return []string{
				"VELOX_DATA_DIR=" + profile,
				"VELOX_BENCH_POLICY_AUDIT=1",
			}
		},
		expected: "ready security-ok\n",
	}
}

func cppHost(t *testing.T, repoRoot string) hostAdapter {
	t.Helper()
	executable := requiredExecutable(t, "VELOX_BUILT_CPP_HOST")
	assets := filepath.Join(repoRoot, "examples", "hello", "web")
	return hostAdapter{
		name:       "cpp23",
		executable: executable,
		arguments: func(profile string) []string {
			return []string{assets, profile}
		},
		environment: func(string) []string { return nil },
		expected:    "ready dom-2raf\n",
	}
}

func requiredExecutable(t *testing.T, environment string) string {
	t.Helper()
	executable := os.Getenv(environment)
	if executable == "" {
		t.Skip(environment + " is set only by its configured startup intent")
	}
	abs, err := filepath.Abs(executable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("built host unavailable: %v", err)
	}
	return abs
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func runHost(t *testing.T, host hostAdapter, profile string) hostRun {
	t.Helper()
	pipeName := fmt.Sprintf(`\\.\pipe\velox-%d`, time.Now().UnixNano())
	pipe := createPipe(t, pipeName)
	defer windows.CloseHandle(pipe)
	defer disconnectNamedPipe.Call(uintptr(pipe))

	cmd := exec.Command(host.executable, host.arguments(profile)...)
	cmd.Env = append(os.Environ(),
		append(host.environment(profile),
			"VELOX_BENCH_PIPE="+pipeName,
			"VELOX_BENCH_EXIT_AFTER_READY=1",
		)...,
	)
	output := &strings.Builder{}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	started := time.Now()

	done := make(chan error, 1)
	go func() {
		if err := acceptPipe(pipe); err != nil {
			done <- err
			return
		}
		buffer := make([]byte, 128)
		n, err := windows.Read(pipe, buffer)
		if err != nil {
			done <- err
			return
		}
		if string(buffer[:n]) != host.expected {
			done <- fmt.Errorf("unexpected marker %q, want %q", buffer[:n], host.expected)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			_ = cmd.Process.Kill()
			t.Fatalf("%s ready marker failed: %v; host output: %s", host.name, err, output.String())
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		cancelIoEx.Call(uintptr(pipe), 0)
		t.Fatalf("%s host did not report ready; output: %s", host.name, output.String())
	}
	readyDuration := time.Since(started)
	exitStarted := time.Now()

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("%s host exit failed: %v; output: %s", host.name, err, output.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("%s host did not exit after ready", host.name)
	}
	return hostRun{Ready: readyDuration, Exit: time.Since(exitStarted)}
}

func summarize(durations []time.Duration) profileResult {
	milliseconds := make([]float64, len(durations))
	for index, duration := range durations {
		milliseconds[index] = float64(duration) / float64(time.Millisecond)
	}
	sorted := append([]float64(nil), milliseconds...)
	sort.Float64s(sorted)
	return profileResult{
		DurationsMS: milliseconds,
		P50MS:       percentile(sorted, 0.50),
		P95MS:       percentile(sorted, 0.95),
	}
}

func percentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	position := percentile * float64(len(sorted)-1)
	lower := int(position)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[lower]
	}
	fraction := position - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*fraction
}

func createPipe(t *testing.T, name string) windows.Handle {
	t.Helper()
	nameUTF16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	handle, _, callErr := createNamedPipeW.Call(
		uintptr(unsafe.Pointer(nameUTF16)),
		pipeAccessInbound,
		pipeTypeByte|pipeWait,
		1,
		4096,
		4096,
		0,
		0,
	)
	if handle == uintptr(windows.InvalidHandle) {
		t.Fatalf("CreateNamedPipeW: %v", callErr)
	}
	return windows.Handle(handle)
}

func acceptPipe(pipe windows.Handle) error {
	connected, _, err := connectNamedPipe.Call(uintptr(pipe), 0)
	if connected != 0 || err == windows.ERROR_PIPE_CONNECTED {
		return nil
	}
	return err
}
