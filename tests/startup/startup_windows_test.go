package startup_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
)

var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	createNamedPipeW    = kernel32.NewProc("CreateNamedPipeW")
	connectNamedPipe    = kernel32.NewProc("ConnectNamedPipe")
	disconnectNamedPipe = kernel32.NewProc("DisconnectNamedPipe")
	cancelIoEx          = kernel32.NewProc("CancelIoEx")
)

type hostAdapter struct {
	name                  string
	executable            string
	arguments             func(profile string) []string
	environment           func(profile string) []string
	expectedPhase         string
	requireBrowserProcess bool
}

type hostRun struct {
	Ready              time.Duration
	Exit               time.Duration
	HostExitedAt       time.Time
	BrowserProcessID   uint32
	BrowserProcessExit <-chan time.Time
}

func TestBuiltHostStartup(t *testing.T) {
	repoRoot := repositoryRoot(t)
	host := goHost(t, repoRoot)
	profile := managedProfileRoot(t, "velox-go-smoke-")
	first := mustRunHost(t, host, profile)
	immediate := mustRunHost(t, host, profile)
	profileReleaseStarted := time.Now()
	profileRelease := mustWaitForProfileRelease(t, profile, 10*time.Second)
	firstBrowserExit := mustAwaitBrowserExit(t, first, 10*time.Second)
	immediateBrowserExit := mustAwaitBrowserExit(t, immediate, 10*time.Second)
	securityProfile := managedProfileRoot(t, "velox-go-security-")
	security := mustRunHost(t, securityHost(t, repoRoot), securityProfile)
	testUnavailableRuntime(t, host, filepath.Join(t.TempDir(), "missing-webview2-runtime"))

	if first.Exit > time.Second || immediate.Exit > time.Second {
		t.Fatalf("host shutdown exceeded 1s: first=%s immediate=%s", first.Exit, immediate.Exit)
	}
	if immediate.Ready > 10*time.Second {
		t.Fatalf("same-profile immediate relaunch exceeded 10s: %s", immediate.Ready)
	}
	t.Logf("first ready=%s host-exit=%s browser-pid=%d browser-exit-after-host=%s; immediate ready=%s host-exit=%s browser-pid=%d browser-exit-after-host=%s; profile-release-wait=%s profile-released-after-immediate-host=%s",
		first.Ready, first.Exit, first.BrowserProcessID, firstBrowserExit,
		immediate.Ready, immediate.Exit, immediate.BrowserProcessID, immediateBrowserExit,
		profileRelease, profileReleaseStarted.Add(profileRelease).Sub(immediate.HostExitedAt))
	t.Logf("security ready=%s exit=%s", security.Ready, security.Exit)
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

func waitForProfileRelease(root string, timeout time.Duration) (time.Duration, error) {
	started := time.Now()
	deadline := started.Add(timeout)
	for {
		err := os.RemoveAll(root)
		if err == nil || os.IsNotExist(err) {
			return time.Since(started), nil
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("WebView2 profile remained locked after %s: %w", timeout, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func mustWaitForProfileRelease(t *testing.T, root string, timeout time.Duration) time.Duration {
	t.Helper()
	duration, err := waitForProfileRelease(root, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return duration
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
		expectedPhase:         "dom-2raf",
		requireBrowserProcess: true,
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
		expectedPhase: "security-ok",
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

func testUnavailableRuntime(t *testing.T, host hostAdapter, missingRuntime string) {
	t.Helper()
	profile := filepath.Join(t.TempDir(), "profile")
	cmd := exec.Command(host.executable, host.arguments(profile)...)
	cmd.Env = append(os.Environ(),
		append(host.environment(profile),
			"VELOX_BENCH_WEBVIEW2_BROWSER_DIR="+missingRuntime,
		)...,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("host started with a missing fixed WebView2 Runtime")
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 5 {
		t.Fatalf("missing-runtime exit = %v; output: %s", err, output)
	}
	const diagnostic = "WebView2 Runtime is unavailable or initialization failed"
	if !strings.Contains(string(output), diagnostic) {
		t.Fatalf("missing-runtime diagnostic = %q, want containing %q", output, diagnostic)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func mustRunHost(t *testing.T, host hostAdapter, profile string) hostRun {
	t.Helper()
	run, err := runHost(host, profile)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func runHost(host hostAdapter, profile string) (hostRun, error) {
	pipeName := fmt.Sprintf(`\\.\pipe\velox-%d`, time.Now().UnixNano())
	pipe, err := createPipe(pipeName)
	if err != nil {
		return hostRun{}, err
	}
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
		return hostRun{}, fmt.Errorf("start %s host: %w", host.name, err)
	}
	started := time.Now()

	type readyResult struct {
		browserProcessID uint32
		browserExit      <-chan time.Time
		err              error
	}
	done := make(chan readyResult, 1)
	go func() {
		if err := acceptPipe(pipe); err != nil {
			done <- readyResult{err: err}
			return
		}
		buffer := make([]byte, 128)
		n, err := windows.Read(pipe, buffer)
		if err != nil {
			done <- readyResult{err: err}
			return
		}
		fields := strings.Fields(string(buffer[:n]))
		if len(fields) < 2 || fields[0] != "ready" || fields[1] != host.expectedPhase {
			done <- readyResult{err: fmt.Errorf("unexpected marker %q, want ready %s", buffer[:n], host.expectedPhase)}
			return
		}
		result := readyResult{}
		if host.requireBrowserProcess {
			if len(fields) != 3 {
				done <- readyResult{err: fmt.Errorf("ready marker %q has no browser process ID", buffer[:n])}
				return
			}
			processID, parseErr := strconv.ParseUint(fields[2], 10, 32)
			if parseErr != nil || processID == 0 {
				done <- readyResult{err: fmt.Errorf("invalid browser process ID %q", fields[2])}
				return
			}
			result.browserProcessID = uint32(processID)
			result.browserExit, result.err = observeProcessExit(result.browserProcessID)
		}
		done <- result
	}()

	var ready readyResult
	select {
	case ready = <-done:
		if ready.err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return hostRun{}, fmt.Errorf("%s ready marker failed: %w; host output: %s", host.name, ready.err, output.String())
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		cancelIoEx.Call(uintptr(pipe), 0)
		_, _ = cmd.Process.Wait()
		return hostRun{}, fmt.Errorf("%s host did not report ready within 15s; output: %s", host.name, output.String())
	}
	readyDuration := time.Since(started)
	exitStarted := time.Now()

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case err := <-waitDone:
		if err != nil {
			return hostRun{}, fmt.Errorf("%s host exit failed: %w; output: %s", host.name, err, output.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-waitDone
		return hostRun{}, fmt.Errorf("%s host did not exit within 5s after ready", host.name)
	}
	hostExitedAt := time.Now()
	return hostRun{
		Ready:              readyDuration,
		Exit:               hostExitedAt.Sub(exitStarted),
		HostExitedAt:       hostExitedAt,
		BrowserProcessID:   ready.browserProcessID,
		BrowserProcessExit: ready.browserExit,
	}, nil
}

func observeProcessExit(processID uint32) (<-chan time.Time, error) {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, processID)
	if err != nil {
		return nil, fmt.Errorf("open browser process %d: %w", processID, err)
	}
	exited := make(chan time.Time, 1)
	go func() {
		defer windows.CloseHandle(handle)
		result, waitErr := windows.WaitForSingleObject(handle, windows.INFINITE)
		if waitErr == nil && result == windows.WAIT_OBJECT_0 {
			exited <- time.Now()
		}
		close(exited)
	}()
	return exited, nil
}

func awaitBrowserExit(run hostRun, timeout time.Duration) (time.Duration, error) {
	if run.BrowserProcessExit == nil {
		return 0, errors.New("browser process exit observation is unavailable")
	}
	select {
	case exitedAt, ok := <-run.BrowserProcessExit:
		if !ok {
			return 0, fmt.Errorf("browser process %d exit observation failed", run.BrowserProcessID)
		}
		return exitedAt.Sub(run.HostExitedAt), nil
	case <-time.After(timeout):
		return 0, fmt.Errorf("browser process %d did not exit within %s", run.BrowserProcessID, timeout)
	}
}

func mustAwaitBrowserExit(t *testing.T, run hostRun, timeout time.Duration) time.Duration {
	t.Helper()
	duration, err := awaitBrowserExit(run, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return duration
}

func createPipe(name string) (windows.Handle, error) {
	nameUTF16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return windows.InvalidHandle, err
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
		return windows.InvalidHandle, fmt.Errorf("CreateNamedPipeW: %w", callErr)
	}
	return windows.Handle(handle), nil
}

func acceptPipe(pipe windows.Handle) error {
	connected, _, err := connectNamedPipe.Call(uintptr(pipe), 0)
	if connected != 0 || err == windows.ERROR_PIPE_CONNECTED {
		return nil
	}
	return err
}
