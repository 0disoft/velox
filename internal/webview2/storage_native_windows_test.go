package webview2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	webview "github.com/jchv/go-webview2"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

const storageProbePrefix = "VELOX_STORAGE_PROBE="

type storageProbeResult struct {
	Status            string `json:"status"`
	BrowserPID        uint32 `json:"browserPid"`
	ProfileLength     int    `json:"profileLength"`
	BrowserPathLength int    `json:"browserPathLength"`
}

// A diagnostic, not a claim that all measured storage operations succeed.
// Bypassing alias resolution exercises its fallback without changing NTFS settings.
func TestNativeStorageWithoutShortAlias(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_STORAGE_PROBE") != "1" {
		t.Skip("opt-in native storage path diagnostic")
	}
	version, err := webviewloader.GetInstalledVersion()
	if err != nil || version == "" {
		t.Fatalf("WebView2 required: %v", err)
	}
	t.Logf("WebView2=%s Go=%s; direct native dependency view, not a packaged host", version, runtime.Version())
	for _, scenario := range []struct {
		name   string
		length int
		alias  bool
	}{
		{"short-fallback", 0, false},
		{"long-resolved", 218, true},
		{"long-fallback", 218, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root, err := os.MkdirTemp("", "velox-storage-probe-")
			if err != nil {
				t.Fatal(err)
			}
			cleanupSafe := true
			t.Cleanup(func() {
				if !cleanupSafe {
					t.Logf("retained unconfirmed browser profile: %s", root)
					return
				}
				if err := os.RemoveAll(root); err != nil {
					t.Errorf("remove isolated profile: %v", err)
				}
			})
			profile := filepath.Join(root, "profile")
			if len(profile) < scenario.length {
				profile = filepath.Join(root, strings.Repeat("p", scenario.length-len(root)-1))
			}
			if err := os.MkdirAll(profile, 0o700); err != nil {
				t.Fatal(err)
			}
			browserPath := browserDataPath(profile)
			if scenario.alias {
				browserPath = resolveBrowserDataPath(profile)
			}
			if scenario.alias && len(browserPath) >= len(browserDataPath(profile)) {
				t.Skip("control requires an existing shorter alias on this volume")
			}
			rounds := []struct {
				mode, path string
				verify     bool
			}{
				{"write", browserPath, scenario.name != "long-fallback"},
				{"read", browserPath, scenario.name != "long-fallback"},
			}
			if scenario.alias {
				rounds = append(rounds,
					struct {
						mode, path string
						verify     bool
					}{"read", browserDataPath(profile), false},
					struct {
						mode, path string
						verify     bool
					}{"read", browserPath, true})
			}
			for _, round := range rounds {
				cleanupSafe = false
				result, exited := runStorageProbeChild(t, profile, round.path, round.mode)
				cleanupSafe = exited
				if !exited {
					t.Fatal("browser exit unconfirmed; profile retained")
				}
				t.Logf("%s: status=%s profile-length=%d browser-path-length=%d", round.mode, result.Status, result.ProfileLength, result.BrowserPathLength)
				if round.verify && result.Status != "verified" {
					t.Fatalf("storage control failed: %+v", result)
				}
				if scenario.name == "long-fallback" && result.Status != "verified" {
					break
				}
			}
		})
	}
}

func runStorageProbeChild(t *testing.T, profile, browserPath, mode string) (storageProbeResult, bool) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestNativeStorageProbeChild$", "-test.v")
	cmd.Env = append(os.Environ(), "VELOX_STORAGE_PROBE_PROFILE="+profile, "VELOX_STORAGE_PROBE_PATH="+browserPath, "VELOX_STORAGE_PROBE_MODE="+mode)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, runErr := cmd.CombinedOutput()
	var result storageProbeResult
	found := false
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, storageProbePrefix) {
			continue
		}
		if found {
			t.Fatal("duplicate probe receipt")
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, storageProbePrefix))), &result); err != nil {
			t.Fatal(err)
		}
		found = true
	}
	exited := false
	if result.BrowserPID != 0 {
		process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, result.BrowserPID)
		if err == windows.ERROR_INVALID_PARAMETER {
			exited = true
		} else if err == nil {
			state, waitErr := windows.WaitForSingleObject(process, 15000)
			windows.CloseHandle(process)
			exited = waitErr == nil && state == windows.WAIT_OBJECT_0
		}
	}
	if runErr != nil || !found || result.Status == "" {
		t.Fatalf("probe failed: %v\n%s", runErr, output)
	}
	return result, exited
}

func TestNativeStorageProbeChild(t *testing.T) {
	profile := os.Getenv("VELOX_STORAGE_PROBE_PROFILE")
	if profile == "" {
		t.Skip("private child of storage diagnostic")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	path := os.Getenv("VELOX_STORAGE_PROBE_PATH")
	mode := os.Getenv("VELOX_STORAGE_PROBE_MODE")
	if mode != "write" && mode != "read" {
		t.Fatal("invalid mode")
	}
	result := storageProbeResult{ProfileLength: len(profile), BrowserPathLength: len(path)}
	view, err := webview.NewWithOptionsAndError(webview.WebViewOptions{
		DataPath: path, DenyAllPermissions: true,
		WindowOptions: webview.WindowOptions{Title: "Velox Isolated Storage Probe", Width: 640, Height: 480},
	})
	if err != nil || view == nil {
		t.Fatalf("create view: %v", err)
	}
	defer view.Destroy()
	result.BrowserPID, err = view.BrowserProcessID()
	if err != nil {
		t.Fatal(err)
	}
	assets := t.TempDir()
	page := `<!doctype html><title>Storage probe</title><p id="status">Checking storage</p><script>
(async () => {
  const expected = "Draft \uD55C\uAE00\r\nSecond line";
  const mode = "` + mode + `";
  const db = await new Promise((resolve, reject) => {
    const request = indexedDB.open("velox-no-alias-probe", 1);
    request.onupgradeneeded = () => request.result.createObjectStore("drafts");
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
    request.onblocked = () => reject(Error("Database blocked"));
  });
  try {
    await new Promise((resolve, reject) => {
      const tx = db.transaction("drafts", mode === "write" ? "readwrite" : "readonly");
      const store = tx.objectStore("drafts");
      const request = mode === "write" ? store.put(expected, "draft") : store.get("draft");
      let verified = mode === "write";
      request.onsuccess = () => { if (mode === "read") verified = request.result === expected; };
      tx.oncomplete = () => verified ? resolve() : reject(Error("Draft mismatch"));
      tx.onabort = () => reject(tx.error || Error("Draft mismatch"));
      tx.onerror = () => reject(tx.error);
    });
  } finally { db.close(); }
  await window.__storageResult("verified");
})().catch(error => window.__storageResult("failed:" + error.name + ":" + error.message));
</script>`
	if err := os.WriteFile(filepath.Join(assets, "index.html"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := view.SetVirtualHostNameToFolderMapping("dev.velox.storage-probe.velox.local", assets); err != nil {
		t.Fatal(err)
	}
	if err := view.Bind("__storageResult", func(status string) {
		result.Status = status
		view.Dispatch(func() { view.Dispatch(view.Destroy) })
	}); err != nil {
		t.Fatal(err)
	}
	view.Navigate("https://dev.velox.storage-probe.velox.local/index.html")
	view.Run()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(storageProbePrefix + string(encoded))
}
