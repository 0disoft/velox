//go:build windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/webview2"
)

func TestStartupFailureReporting(t *testing.T) {
	for _, headless := range []bool{false, true} {
		for _, kind := range []startupFailureKind{startupHost, startupConfiguration, startupProfile, startupRuntime, startupDisplay, startupInstance} {
			var output bytes.Buffer
			var notices []startupFailureKind
			reporter := startupFailureReporter{output: &output, headless: headless, notice: func(k startupFailureKind) { notices = append(notices, k) }}
			err := errors.New(`private C:\Users\person\secret.txt document content`)
			if code := reporter.fail(kind, err, 6); code != 6 || output.String() != "velox-host: "+err.Error()+"\n" {
				t.Fatalf("stderr or exit changed: %d %q", code, output.String())
			}
			if headless && len(notices) != 0 || !headless && (len(notices) != 1 || notices[0] != kind) {
				t.Fatalf("headless=%t notices=%v", headless, notices)
			}
			message := startupFailureMessage(kind)
			if strings.Contains(message, "private") || strings.Contains(message, `C:\`) || strings.Contains(message, "document content") || len(message) > 400 {
				t.Fatalf("unsafe or unbounded notice: %q", message)
			}
		}
	}
}

func TestRuntimeFailurePreservesCancellationAndExitCodes(t *testing.T) {
	for _, tt := range []struct {
		err  error
		code int
	}{
		{fmt.Errorf("wrapped: %w", webview2.ErrInitializationCanceled), 0},
		{fmt.Errorf("wrapped: %w", webview2.ErrRuntimeUnavailable), 5},
		{errors.New("bind bridge failed"), 6},
	} {
		var output bytes.Buffer
		calls := 0
		reporter := startupFailureReporter{output: &output, notice: func(kind startupFailureKind) {
			calls++
			if kind != startupRuntime {
				t.Errorf("kind=%d", kind)
			}
		}}
		if got := reporter.runtimeFailure(tt.err); got != tt.code {
			t.Fatalf("code=%d want %d", got, tt.code)
		}
		if tt.code == 0 {
			if calls != 0 || output.Len() != 0 {
				t.Fatal("user cancellation emitted a failure")
			}
		} else if calls != 1 || output.String() != "velox-host: "+tt.err.Error()+"\n" {
			t.Fatal("runtime failure not reported exactly once")
		}
	}
}

func TestStartupNoticeForInvalidConfigurationAndUsage(t *testing.T) {
	t.Setenv("VELOX_BENCH_MODE", "")
	for _, args := range [][]string{
		{"--config", filepath.Join(t.TempDir(), "missing.json")},
		{"--unknown"},
	} {
		calls := 0
		code := runWithStartupNotice(args, func(kind startupFailureKind) {
			calls++
			if kind != startupConfiguration {
				t.Errorf("kind=%d", kind)
			}
		})
		if code != 2 || calls != 1 {
			t.Fatalf("code=%d calls=%d", code, calls)
		}
	}
	calls := 0
	if code := runWithStartupNotice([]string{"--help"}, func(startupFailureKind) { calls++ }); code != 2 || calls != 0 {
		t.Fatal("help showed a failure notice")
	}
	t.Setenv("VELOX_BENCH_MODE", "1")
	runWithStartupNotice([]string{"--unknown"}, func(startupFailureKind) { calls++ })
	if calls != 0 {
		t.Fatal("benchmark showed a modal notice")
	}
}

func TestStartupProfileFailureDoesNotResetData(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "web"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web", "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "velox.runtime.json")
	if err := os.WriteFile(config, []byte(`{"runtimeVersion":1,"app":{"id":"dev.velox.startup-test","name":"Test","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(root, "profile")
	const original = "existing private data"
	if err := os.WriteFile(profile, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VELOX_BENCH_MODE", "")
	t.Setenv("VELOX_DATA_DIR", profile)
	calls := 0
	code := runWithStartupNotice([]string{"--config", config}, func(kind startupFailureKind) {
		calls++
		if kind != startupProfile {
			t.Errorf("kind=%d", kind)
		}
	})
	if code != 5 || calls != 1 {
		t.Fatalf("code=%d calls=%d", code, calls)
	}
	body, err := os.ReadFile(profile)
	if err != nil || string(body) != original {
		t.Fatalf("existing data changed: %q %v", body, err)
	}
	relative := filepath.Base(root) + "-relative"
	t.Setenv("VELOX_DATA_DIR", relative)
	calls = 0
	code = runWithStartupNotice([]string{"--config", config}, func(kind startupFailureKind) {
		calls++
		if kind != startupProfile {
			t.Errorf("kind=%d", kind)
		}
	})
	if code != 6 || calls != 1 {
		t.Fatalf("relative profile code=%d calls=%d", code, calls)
	}
	if _, err := os.Stat(relative); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("relative profile was created: %v", err)
	}
}
