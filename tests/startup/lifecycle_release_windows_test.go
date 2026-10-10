package startup_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

type lifecycleRelease struct {
	StartedAt                time.Time
	FirstBrowserExitedAt     time.Time
	ImmediateBrowserExitedAt time.Time
	ProfileReleasedAt        time.Time
}

type lifecycleReleaseFailure struct {
	Phase string
	Code  string
	Cause error
}

func (failure *lifecycleReleaseFailure) Error() string {
	return fmt.Sprintf("%s: %v", failure.Phase, failure.Cause)
}

// Browser observation and post-exit removal share one deadline. In particular,
// a failed browser observation must not partially delete a still-live profile.
func waitForLifecycleRelease(profile string, first, immediate hostRun, budget time.Duration) (lifecycleRelease, *lifecycleReleaseFailure) {
	result := lifecycleRelease{StartedAt: time.Now()}
	deadline := result.StartedAt.Add(budget)
	for index, run := range []hostRun{first, immediate} {
		phase := "first-browser-exit"
		if index == 1 {
			phase = "immediate-browser-exit"
		}
		exitedAt, err := awaitBrowserExitAt(run, time.Until(deadline))
		if err == nil && exitedAt.After(deadline) {
			err = fmt.Errorf("browser process %d exited after the shared %s deadline", run.BrowserProcessID, budget)
		}
		if err != nil {
			return result, &lifecycleReleaseFailure{Phase: phase, Code: "BROWSER_EXIT_FAILED", Cause: err}
		}
		if index == 0 {
			result.FirstBrowserExitedAt = exitedAt
		} else {
			result.ImmediateBrowserExitedAt = exitedAt
		}
	}
	started := time.Now()
	elapsed, err := waitForProfileRelease(profile, time.Until(deadline))
	if err == nil && started.Add(elapsed).After(deadline) {
		err = fmt.Errorf("profile removal completed after the shared %s deadline", budget)
	}
	if err != nil {
		return result, &lifecycleReleaseFailure{Phase: "profile-release", Code: "PROFILE_RELEASE_FAILED", Cause: err}
	}
	result.ProfileReleasedAt = started.Add(elapsed)
	return result, nil
}

func exitedBrowser(at time.Time) hostRun {
	exit := make(chan time.Time, 1)
	exit <- at
	close(exit)
	return hostRun{BrowserProcessID: 1, BrowserProcessExit: exit}
}

func TestLifecycleReleaseOrder(t *testing.T) {
	for _, scenario := range []string{"first-unobserved", "immediate-unobserved", "failed-observation", "success"} {
		t.Run(scenario, func(t *testing.T) {
			profile := t.TempDir()
			marker := filepath.Join(profile, "keep.txt")
			if err := os.WriteFile(marker, []byte("preserve while browser is alive"), 0o600); err != nil {
				t.Fatal(err)
			}
			first, second := exitedBrowser(time.Now()), exitedBrowser(time.Now())
			expectedPhase := ""
			switch scenario {
			case "first-unobserved":
				first.BrowserProcessExit = make(chan time.Time)
				expectedPhase = "first-browser-exit"
			case "immediate-unobserved":
				second.BrowserProcessExit = make(chan time.Time)
				expectedPhase = "immediate-browser-exit"
			case "failed-observation":
				exit := make(chan time.Time)
				close(exit)
				first.BrowserProcessExit = exit
				expectedPhase = "first-browser-exit"
			}
			result, failure := waitForLifecycleRelease(profile, first, second, 30*time.Millisecond)
			if expectedPhase != "" {
				if failure == nil || failure.Phase != expectedPhase || failure.Code != "BROWSER_EXIT_FAILED" {
					t.Fatalf("failure = %+v, want %s", failure, expectedPhase)
				}
				if body, err := os.ReadFile(marker); err != nil || string(body) != "preserve while browser is alive" {
					t.Fatalf("live profile changed: %q, %v", body, err)
				}
				return
			}
			if failure != nil || result.FirstBrowserExitedAt.IsZero() || result.ImmediateBrowserExitedAt.IsZero() || result.ProfileReleasedAt.IsZero() {
				t.Fatalf("incomplete release: %+v, %v", result, failure)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("profile was not removed after both browsers exited: %v", err)
			}
		})
	}
}

func TestLifecycleReleasePostExitLock(t *testing.T) {
	profile := t.TempDir()
	path := filepath.Join(profile, "lockfile")
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	result, failure := waitForLifecycleRelease(profile, exitedBrowser(time.Now()), exitedBrowser(time.Now()), 30*time.Millisecond)
	if failure == nil || failure.Phase != "profile-release" || failure.Code != "PROFILE_RELEASE_FAILED" {
		t.Fatalf("post-exit lock not distinguished: %+v", failure)
	}
	if result.FirstBrowserExitedAt.IsZero() || result.ImmediateBrowserExitedAt.IsZero() {
		t.Fatal("browser observations lost on profile failure")
	}
}

func TestLifecycleReleaseSharedBudget(t *testing.T) {
	profile := t.TempDir()
	marker := filepath.Join(profile, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, second := make(chan time.Time, 1), make(chan time.Time, 1)
	firstTimer := time.AfterFunc(100*time.Millisecond, func() { first <- time.Now() })
	secondTimer := time.AfterFunc(350*time.Millisecond, func() { second <- time.Now() })
	defer firstTimer.Stop()
	defer secondTimer.Stop()
	result, failure := waitForLifecycleRelease(profile,
		hostRun{BrowserProcessExit: first}, hostRun{BrowserProcessExit: second}, 300*time.Millisecond)
	if failure == nil || failure.Phase != "immediate-browser-exit" || result.FirstBrowserExitedAt.IsZero() {
		t.Fatalf("browser waits did not share the deadline: %+v, %v", result, failure)
	}
	if body, err := os.ReadFile(marker); err != nil || string(body) != "keep" {
		t.Fatalf("shared deadline removed live profile: %q, %v", body, err)
	}
}

func TestLifecycleReleaseDeadline(t *testing.T) {
	profile := t.TempDir()
	marker := filepath.Join(profile, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, failure := waitForLifecycleRelease(profile, exitedBrowser(time.Now().Add(time.Second)), exitedBrowser(time.Now()), 30*time.Millisecond)
	if failure == nil || failure.Phase != "first-browser-exit" {
		t.Fatalf("late observation accepted: %+v", failure)
	}
	_, failure = waitForLifecycleRelease(profile, exitedBrowser(time.Now()), exitedBrowser(time.Now().Add(time.Second)), 30*time.Millisecond)
	if failure == nil || failure.Phase != "immediate-browser-exit" {
		t.Fatalf("second browser received a fresh deadline: %+v", failure)
	}
	if _, err := waitForProfileRelease(profile, 0); err == nil {
		t.Fatal("expired removal budget accepted")
	}
	if body, err := os.ReadFile(marker); err != nil || string(body) != "keep" {
		t.Fatalf("expired budget deleted profile: %q, %v", body, err)
	}
}

func TestLifecycleProfileCleanupGuard(t *testing.T) {
	var root string
	t.Run("unconfirmed-browser", func(t *testing.T) {
		root = managedProfileRoot(t, "velox-cleanup-guard-", func() bool { return false })
		if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	defer os.RemoveAll(root)
	if body, err := os.ReadFile(filepath.Join(root, "keep.txt")); err != nil || string(body) != "keep" {
		t.Fatalf("cleanup modified unconfirmed profile: %q, %v", body, err)
	}
	t.Run("confirmed-browser", func(t *testing.T) {
		root = managedProfileRoot(t, "velox-cleanup-guard-", func() bool { return true })
	})
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("confirmed profile was not cleaned: %v", err)
	}
}
