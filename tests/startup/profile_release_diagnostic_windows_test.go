package startup_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	veloxwebview2 "github.com/0disoft/velox/internal/webview2"
	"golang.org/x/sys/windows"
)

type profileReleaseDiagnostic struct {
	SchemaVersion string                `json:"schemaVersion"`
	SourceCommit  string                `json:"sourceCommit"`
	HostSHA256    string                `json:"hostSha256"`
	WebView2      string                `json:"webview2Version"`
	StartedAtUTC  time.Time             `json:"startedAtUtc"`
	Trials        []profileReleaseTrial `json:"trials"`
}

type profileReleaseTrial struct {
	Mode                  string             `json:"mode"`
	FirstReadyMs          float64            `json:"firstReadyMs"`
	SecondReadyMs         float64            `json:"secondReadyMs"`
	FirstExitMs           float64            `json:"firstHostExitMs"`
	SecondExitMs          float64            `json:"secondHostExitMs"`
	FirstBrowserPID       uint32             `json:"firstBrowserPid"`
	SecondBrowserPID      uint32             `json:"secondBrowserPid"`
	FirstBrowserExitMs    *float64           `json:"firstBrowserExitAfterHostMs"`
	SecondBrowserExitMs   *float64           `json:"secondBrowserExitAfterHostMs"`
	SecondReadyAfterFirst *float64           `json:"secondReadyAfterFirstBrowserExitMs"`
	LockDeleteAccessMs    *float64           `json:"lockDeleteAccessAfterSecondHostMs"`
	DirectoryRemovedMs    *float64           `json:"directoryRemovedAfterSecondHostMs"`
	FilesBefore           int                `json:"filesBeforeObservation"`
	FilesAfterFirstRemove *int               `json:"filesAfterFirstRemove"`
	InitialRemoveError    string             `json:"initialRemoveError,omitempty"`
	LockErrors            map[string]int     `json:"lockProbeErrors"`
	LockOwners            []profileLockOwner `json:"lockOwners,omitempty"`
	OwnerQueryMs          *float64           `json:"ownerQueryMs,omitempty"`
	OwnerQueryError       string             `json:"ownerQueryError,omitempty"`
	SecondShutdown        *json.RawMessage   `json:"secondShutdownTimeline,omitempty"`
	Error                 string             `json:"error,omitempty"`
}

// This deliberately separate diagnostic leaves the production host unchanged.
// Only the legacy comparison arm deletes files while the browser is still live.
func TestProfileReleaseDiagnostic(t *testing.T) {
	runProfileReleaseDiagnostic(t, "VELOX_PROFILE_RELEASE_DIAGNOSTIC_RESULT",
		[]string{"legacy-delete-first", "browser-exit-first", "browser-exit-first", "legacy-delete-first"})
}

func TestProfileLockOwnerDiagnostic(t *testing.T) {
	evidence := runProfileReleaseDiagnostic(t, "VELOX_PROFILE_LOCK_OWNER_DIAGNOSTIC_RESULT",
		[]string{"browser-exit-first-owner-query"})
	trial := evidence.Trials[0]
	if trial.OwnerQueryError != "" {
		t.Fatal(trial.OwnerQueryError)
	}
	for _, owner := range trial.LockOwners {
		if owner.PID == trial.SecondBrowserPID && owner.StartedAt != 0 {
			return
		}
	}
	t.Fatalf("browser PID %d not identified as lock owner: %+v", trial.SecondBrowserPID, trial.LockOwners)
}

func runProfileReleaseDiagnostic(t *testing.T, resultEnv string, modes []string) profileReleaseDiagnostic {
	t.Helper()
	resultPath := os.Getenv(resultEnv)
	if resultPath == "" {
		t.Skip("profile-release diagnostic requires an explicit result path")
	}
	host := goHost(t, repositoryRoot(t))
	body, err := os.ReadFile(host.executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	version, err := veloxwebview2.InstalledVersion()
	if err != nil {
		t.Fatal(err)
	}
	evidence := profileReleaseDiagnostic{
		SchemaVersion: "velox.profile-release-diagnostic/v1",
		SourceCommit:  os.Getenv("VELOX_DIAGNOSTIC_SOURCE_COMMIT"),
		HostSHA256:    hex.EncodeToString(digest[:]), WebView2: version,
		StartedAtUTC: time.Now().UTC(),
	}
	// Counterbalance the order without an unbounded stress loop.
	for _, mode := range modes {
		trial := measureProfileRelease(t, host, mode)
		evidence.Trials = append(evidence.Trials, trial)
		t.Logf("mode=%s second-ready=%.2fms browser-exit=%s lock-access=%s removed=%s files=%d error=%s",
			mode, trial.SecondReadyMs, diagnosticMilliseconds(trial.SecondBrowserExitMs),
			diagnosticMilliseconds(trial.LockDeleteAccessMs), diagnosticMilliseconds(trial.DirectoryRemovedMs),
			trial.FilesBefore, trial.Error)
	}
	body, err = json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(resultPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, trial := range evidence.Trials {
		if trial.Error != "" {
			t.Errorf("%s: %s (evidence: %s)", trial.Mode, trial.Error, resultPath)
		}
	}
	return evidence
}

func measureProfileRelease(t *testing.T, host hostAdapter, mode string) profileReleaseTrial {
	profile := managedProfileRoot(t, "velox-release-diagnostic-")
	trial := profileReleaseTrial{Mode: mode, LockErrors: make(map[string]int)}
	first, err := runHost(host, profile)
	if err != nil {
		trial.Error = err.Error()
		return trial
	}
	second, err := runHost(host, profile)
	if err != nil {
		_, _ = awaitBrowserExit(first, 10*time.Second)
		trial.Error = err.Error()
		return trial
	}
	trial.FirstReadyMs, trial.SecondReadyMs = milliseconds(first.Ready), milliseconds(second.Ready)
	trial.FirstExitMs, trial.SecondExitMs = milliseconds(first.Exit), milliseconds(second.Exit)
	trial.FirstBrowserPID, trial.SecondBrowserPID = first.BrowserProcessID, second.BrowserProcessID
	if body, err := json.Marshal(second.ShutdownTimeline); err == nil {
		raw := json.RawMessage(body)
		trial.SecondShutdown = &raw
	}
	trial.FilesBefore, err = countProfileFiles(profile)
	if err != nil {
		trial.Error = err.Error()
		return trial
	}
	firstExit, secondExit := first.BrowserProcessExit, second.BrowserProcessExit
	lockPath := filepath.Join(profile, "EBWebView", "lockfile")
	if mode == "browser-exit-first-owner-query" {
		started := time.Now()
		trial.LockOwners, err = queryProfileLockOwners(lockPath)
		trial.OwnerQueryMs = msPointer(time.Since(started))
		if err != nil {
			trial.OwnerQueryError = err.Error()
		}
	}
	deadline := second.HostExitedAt.Add(20 * time.Second)
	removeAttempted := false
	for {
		select {
		case at, ok := <-firstExit:
			firstExit = nil
			if ok {
				trial.FirstBrowserExitMs = msPointer(at.Sub(first.HostExitedAt))
				trial.SecondReadyAfterFirst = msPointer(second.ReadyAt.Sub(at))
			}
		default:
		}
		select {
		case at, ok := <-secondExit:
			secondExit = nil
			if ok {
				trial.SecondBrowserExitMs = msPointer(at.Sub(second.HostExitedAt))
			}
		default:
		}
		if trial.LockDeleteAccessMs == nil && trial.DirectoryRemovedMs == nil {
			if err := probeProfileDeleteAccess(lockPath); err == nil {
				trial.LockDeleteAccessMs = msPointer(time.Since(second.HostExitedAt))
			} else {
				trial.LockErrors[err.Error()]++
			}
		}
		if trial.DirectoryRemovedMs == nil && (mode == "legacy-delete-first" || (firstExit == nil && secondExit == nil)) {
			err := os.RemoveAll(profile)
			if !removeAttempted && mode == "legacy-delete-first" {
				removeAttempted = true
				if err != nil {
					trial.InitialRemoveError = err.Error()
				}
				remaining, scanErr := countProfileFiles(profile)
				if scanErr == nil {
					trial.FilesAfterFirstRemove = &remaining
				}
			}
			if err == nil {
				trial.DirectoryRemovedMs = msPointer(time.Since(second.HostExitedAt))
			}
		}
		if firstExit == nil && secondExit == nil && trial.DirectoryRemovedMs != nil {
			break
		}
		if time.Now().After(deadline) {
			trial.Error = "browser exit or profile removal exceeded diagnostic 20s observation window"
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if trial.FirstBrowserExitMs == nil || trial.SecondBrowserExitMs == nil {
		trial.Error = "browser process exit observation unavailable or incomplete"
	}
	return trial
}

func msPointer(duration time.Duration) *float64 {
	value := milliseconds(duration)
	return &value
}

func diagnosticMilliseconds(value *float64) string {
	if value == nil {
		return "unobserved"
	}
	return fmt.Sprintf("%.2fms", *value)
}

// Opening with DELETE access checks delete-sharing without deleting or
// truncating anything. A missing lockfile is not evidence of a free handle.
func probeProfileDeleteAccess(path string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(name, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return err
	}
	return windows.CloseHandle(handle)
}

func countProfileFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			count++
		}
		return nil
	})
	return count, err
}

func TestProfileDeleteAccessProbe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lockfile")
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(handle) })
	if err := probeProfileDeleteAccess(path); !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("locked probe = %v, want sharing violation", err)
	}
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = windows.InvalidHandle
	if err := os.WriteFile(path, []byte("preserve contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := probeProfileDeleteAccess(path); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "preserve contents" {
		t.Fatalf("probe modified file: %q, %v", body, err)
	}
	if err := probeProfileDeleteAccess(path + ".missing"); !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		t.Fatal(fmt.Errorf("missing lockfile probe: %w", err))
	}
}
