package webview2

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeRuntimeProfileStorage(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_STORAGE_PROBE") != "1" {
		t.Skip("opt-in real Runtime.Open storage lifecycle")
	}
	testRuntimeProfileStorage(t, false, false)
}

func TestNativeRuntimeProfileExistingDraft(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_STORAGE_PROBE") != "1" {
		t.Skip("opt-in existing dependency-view draft recovery by Runtime.Open")
	}
	testRuntimeProfileStorage(t, true, false)
}

func TestNativeRuntimeTerminateStorage(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_STORAGE_PROBE") != "1" {
		t.Skip("opt-in real runtime termination and committed draft recovery")
	}
	testRuntimeProfileStorage(t, false, true)
}

func testRuntimeProfileStorage(t *testing.T, existingDraft, terminate bool) {
	t.Helper()
	root, err := os.MkdirTemp("", "vpr-")
	if err != nil {
		t.Fatal(err)
	}
	cleanupSafe := true
	t.Cleanup(func() {
		if !cleanupSafe {
			t.Logf("retained fixture with unconfirmed browser exit: %s", root)
			return
		}
		if filepath.Dir(root) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(root), "vpr-") {
			t.Error("unsafe fixture cleanup path")
			return
		}
		removeFixtureProfileLinks(t, filepath.Join(root, "VeloxProfileLinks"))
		removeFixtureProfileLinks(t, filepath.Join(root, "seed"))
		if t.Failed() {
			return
		}
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	profile := root
	// Multiple 8-character segments keep even an existing 8.3 spelling long.
	for range 22 {
		profile = filepath.Join(profile, "p0000000")
	}
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if len(resolveBrowserDataPath(profile)) <= maxProfileLinkPath {
		t.Fatal("fixture must exercise production junction selection")
	}
	marker := filepath.Join(profile, "marker.txt")
	if err := os.WriteFile(marker, []byte("existing profile"), 0o600); err != nil {
		t.Fatal(err)
	}
	modes := []string{"write", "read"}
	if existingDraft {
		lease, err := acquireProfileLink(profile, filepath.Join(root, "seed"))
		if err != nil {
			t.Fatal(err)
		}
		defer lease.Close()
		cleanupSafe = false
		seeded, exited := runStorageProbeChild(t, profile, shortBrowserProfilePath(lease.path), "production-write")
		cleanupSafe = exited
		if !exited || seeded.Status != "verified" || seeded.Origin != trustedOrigin("dev.velox.storage-probe") {
			t.Fatalf("existing dependency-view draft not committed: %+v exited=%v", seeded, exited)
		}
		if err := lease.Close(); err != nil {
			t.Fatal(err)
		}
		t.Log("preexisting Korean/CRLF draft committed by dependency view using a different same-directory spelling")
		modes = []string{"read", "read"}
	}
	for _, mode := range modes {
		cleanupSafe = false
		probeMode := mode
		if terminate {
			probeMode = "terminate-" + mode
		}
		result, exited := runStorageProbeChild(t, profile, root, "runtime-"+probeMode)
		cleanupSafe = exited
		if !exited || result.Status != "verified" || result.BrowserPathLength > maxProfileLinkPath {
			t.Fatalf("runtime storage %s: %+v exited=%v", mode, result, exited)
		}
		t.Logf("Runtime.Open %s: profile-length=%d browser-path-length=%d; browser exited", mode, result.ProfileLength, result.BrowserPathLength)
	}
	if body, err := os.ReadFile(marker); err != nil || string(body) != "existing profile" {
		t.Fatalf("original profile changed: %q %v", body, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "VeloxProfileLinks"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("relaunch did not reuse one stable link: %v %v", entries, err)
	}
}

func runRuntimeStorageProbeChild(t *testing.T, profile, cacheDir, mode string) {
	t.Helper()
	terminate := strings.HasPrefix(mode, "terminate-")
	mode = strings.TrimPrefix(mode, "terminate-")
	if mode != "write" && mode != "read" {
		t.Fatal("invalid runtime probe mode")
	}
	t.Setenv("LOCALAPPDATA", cacheDir)
	assets := t.TempDir()
	entry := filepath.Join(assets, "index.html")
	if err := os.WriteFile(entry, []byte(storageProbePage(mode, "__veloxReady")), 0o600); err != nil {
		t.Fatal(err)
	}
	result := storageProbeResult{ProfileLength: len(profile), Origin: trustedOrigin("dev.velox.storage-probe")}
	var view *Runtime
	view, err := Open(Config{Title: "Velox Isolated Runtime Storage Probe", AppID: "dev.velox.storage-probe", AppVersion: "1",
		Width: 640, Height: 480, DataPath: profile, AssetRoot: assets, EntryPath: entry}, func(status string) error {
		result.Status = status
		if terminate {
			view.Terminate()
		} else {
			view.Close()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.profileLease == nil {
		view.Close()
		view.Run()
		t.Fatal("production Runtime.Open did not select the guarded link")
	}
	linkPath := view.profileLease.path
	result.BrowserPathLength = len(linkPath)
	result.BrowserPID, err = view.BrowserProcessID()
	if err != nil {
		view.Close()
		view.Run()
		t.Fatal(err)
	}
	view.Run()
	view.Close()
	if _, err := view.BrowserProcessID(); err == nil {
		t.Fatal("runtime loop returned with its browser interface retained")
	}
	if len(view.profileLease.handles) != 0 {
		t.Fatal("run-loop exit did not release the profile lease")
	}
	if _, err := os.Stat(linkPath); err != nil {
		t.Fatalf("published link must persist after exit: %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(storageProbePrefix + string(encoded))
}

func TestRuntimeProfilePreparationFailureDoesNotOpenView(t *testing.T) {
	assets := t.TempDir()
	entry := filepath.Join(assets, "index.html")
	if err := os.WriteFile(entry, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	started := false
	view, err := openWithProfilePath(Config{Title: "Test", AppID: "dev.velox.profile-test", AppVersion: "1",
		Width: 640, Height: 480, DataPath: assets, AssetRoot: assets, EntryPath: entry,
		StartupPhase: func(string) { started = true }}, func(string) error { return nil },
		func(string) (string, *profileLink, error) { return "", nil, ErrProfileUnavailable })
	if view != nil || !errors.Is(err, ErrProfileUnavailable) || started {
		t.Fatalf("profile failure continued startup: %v %v started=%v", view, err, started)
	}
}

func TestNativeRuntimeProfileStartupFailureReleasesLease(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_STORAGE_PROBE") != "1" {
		t.Skip("opt-in real Runtime.Open startup failure")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	fixture, profile, root := profileLinkFixture(t)
	lease, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	entry := filepath.Join(fixture, "index.html")
	if err := os.WriteFile(entry, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	missingRuntime := filepath.Join(fixture, "missing-runtime")
	t.Setenv("WEBVIEW2_BROWSER_EXECUTABLE_FOLDER", missingRuntime)
	view, err := openWithProfilePath(Config{Title: "Velox Isolated Startup Failure Probe", AppID: "dev.velox.profile-test", AppVersion: "1",
		Width: 640, Height: 480, DataPath: profile, AssetRoot: fixture, EntryPath: entry,
		BrowserExecutableFolder: missingRuntime}, func(string) error { return nil },
		func(string) (string, *profileLink, error) { return lease.path, lease, nil })
	if view != nil {
		view.Close()
		view.Run()
		t.Fatal("missing fixed runtime unexpectedly opened a view")
	}
	if !errors.Is(err, ErrRuntimeUnavailable) || len(lease.handles) != 0 {
		t.Fatalf("startup failure leaked lease: %v handles=%d", err, len(lease.handles))
	}
	if err := os.Remove(lease.path); err != nil {
		t.Fatalf("failed initialization left the alias pinned: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(profile, "marker.txt")); err != nil || string(body) != "original fixture" {
		t.Fatalf("failed startup changed data: %q %v", body, err)
	}
}
