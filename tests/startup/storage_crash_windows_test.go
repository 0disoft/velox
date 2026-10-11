package startup_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestBuiltHostGuardedStorageAfterAbruptExit(t *testing.T) {
	if os.Getenv("VELOX_STORAGE_PATH_TEST") != "1" {
		t.Skip("opt-in real WebView2 committed-draft crash recovery")
	}
	executable := requiredExecutable(t, "VELOX_BUILT_HOST")
	for _, stopBrowser := range []bool{false, true} {
		t.Run(fmt.Sprintf("stop-browser-%t", stopBrowser), func(t *testing.T) {
			root, err := os.MkdirTemp("", "vpc-")
			if err != nil {
				t.Fatal(err)
			}
			cleanupSafe := true
			t.Cleanup(func() {
				if !cleanupSafe {
					t.Logf("retained isolated crash fixture pending browser exit: %s", root)
					return
				}
				links := filepath.Join(root, "VeloxProfileLinks")
				entries, err := os.ReadDir(links)
				if err != nil && !os.IsNotExist(err) {
					t.Error(err)
					return
				}
				for _, entry := range entries {
					path := filepath.Join(links, entry.Name())
					name, _ := windows.UTF16PtrFromString(path)
					attributes, err := windows.GetFileAttributes(name)
					if err != nil || attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
						t.Errorf("unexpected crash fixture link: %v", err)
						return
					}
					if err := os.Remove(path); err != nil {
						t.Error(err)
						return
					}
				}
				if filepath.Dir(root) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(root), "vpc-") {
					t.Error("unsafe crash fixture cleanup path")
					return
				}
				if err := os.RemoveAll(root); err != nil {
					t.Error(err)
				}
			})
			profile := root
			for range 22 {
				profile = filepath.Join(profile, "p0000000")
			}
			if err := os.MkdirAll(profile, 0o700); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(profile, "marker.txt")
			if err := os.WriteFile(marker, []byte("original isolated profile"), 0o600); err != nil {
				t.Fatal(err)
			}
			assets := filepath.Join(root, "web")
			if err := os.Mkdir(assets, 0o700); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"write", "read"} {
				if err := os.WriteFile(filepath.Join(assets, mode+".html"), []byte(crashStoragePage(mode)), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			configPath := filepath.Join(root, "velox.runtime.json")
			host := hostAdapter{name: "guarded-storage-crash", executable: executable,
				arguments:     func(string) []string { return []string{"--config", configPath} },
				environment:   func(profile string) []string { return []string{"VELOX_DATA_DIR=" + profile, "LOCALAPPDATA=" + root} },
				expectedPhase: "dom-2raf", requireBrowserProcess: true}
			configure := func(mode string) {
				config := map[string]any{"runtimeVersion": 1,
					"app":      map[string]any{"id": "dev.velox.storage-crash", "name": "Velox Isolated Crash Recovery", "version": "1"},
					"assets":   map[string]any{"root": "web", "entry": mode + ".html"},
					"window":   map[string]any{"width": 640, "height": 480},
					"security": map[string]any{"permissions": []string{}}}
				data, err := json.Marshal(config)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(configPath, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			configure("write")
			cleanupSafe = false
			written, err := runHostWithStop(host, profile, func(process *os.Process, browserPID uint32) error {
				browser, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, browserPID)
				if err != nil {
					return err
				}
				defer windows.CloseHandle(browser)
				if state, err := windows.WaitForSingleObject(browser, 0); err != nil || state != uint32(windows.WAIT_TIMEOUT) {
					return fmt.Errorf("owned browser not alive at committed-draft receipt: state=%d error=%v", state, err)
				}
				if err := process.Kill(); err != nil {
					return err
				}
				if stopBrowser {
					return windows.TerminateProcess(browser, 23)
				}
				return nil
			})
			if err != nil || !written.AbruptExit {
				t.Fatalf("crash write did not confirm abrupt exit: %v", err)
			}
			exit, err := awaitBrowserExit(written, 15*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			cleanupSafe = true
			entries, err := os.ReadDir(filepath.Join(root, "VeloxProfileLinks"))
			if err != nil || len(entries) != 1 {
				t.Fatalf("stable link did not survive host interruption: %v %v", entries, err)
			}
			linkName := entries[0].Name()
			// DELETE access checks that the killed host no longer pins its link;
			// the test does not rename or delete it before recovery.
			name, _ := windows.UTF16PtrFromString(filepath.Join(root, "VeloxProfileLinks", linkName))
			handle, err := windows.CreateFile(name, windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
				nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
			if err != nil {
				t.Fatalf("killed host's link lease was not released: %v", err)
			}
			windows.CloseHandle(handle)
			configure("read")
			cleanupSafe = false
			recovered := mustRunHost(t, host, profile)
			mustAwaitBrowserExit(t, recovered, 15*time.Second)
			cleanupSafe = true
			entries, err = os.ReadDir(filepath.Join(root, "VeloxProfileLinks"))
			if err != nil || len(entries) != 1 || entries[0].Name() != linkName {
				t.Fatalf("recovery changed link identity: %v %v", entries, err)
			}
			if body, err := os.ReadFile(marker); err != nil || string(body) != "original isolated profile" {
				t.Fatalf("recovery changed original marker: %q %v", body, err)
			}
			t.Logf("committed Korean/CRLF draft recovered; stop-browser=%t host-abrupt=true browser-exit-after-host=%s profile-length=%d stable-link=%s", stopBrowser, exit, len(profile), linkName)
		})
	}
}

func crashStoragePage(mode string) string {
	return `<!doctype html><title>Isolated crash recovery</title><p>Checking committed draft</p><script>
(async () => {
  const mode = "` + mode + `";
  const expected = "Committed \uD55C\uAE00\r\nSecond line";
  const db = await new Promise((resolve, reject) => {
    const request = indexedDB.open("velox-crash-recovery", 1);
    request.onupgradeneeded = () => request.result.createObjectStore("drafts");
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
  });
  await new Promise((resolve, reject) => {
    const tx = db.transaction("drafts", mode === "write" ? "readwrite" : "readonly");
    const store = tx.objectStore("drafts");
    let matches = mode === "write";
    const request = mode === "write" ? store.put(expected, "draft") : store.get("draft");
    request.onsuccess = () => { if (mode === "read") matches = request.result === expected; };
    tx.oncomplete = () => matches ? resolve() : reject(Error("Draft mismatch"));
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error || Error("Aborted"));
  });
  // Keep the database and browser alive; the parent kills only owned processes.
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  await window.__veloxReady("dom-2raf");
})().catch(error => { document.querySelector("p").textContent = "Failed: " + error.name; });
</script>`
}
