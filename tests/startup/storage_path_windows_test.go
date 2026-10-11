package startup_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This opt-in test requires real WebView2 and never uses an installed app profile.
func TestBuiltHostIndexedDBLongPath(t *testing.T) {
	if os.Getenv("VELOX_STORAGE_PATH_TEST") != "1" {
		t.Skip("VELOX_STORAGE_PATH_TEST=1 enables the native long-profile storage check")
	}
	executable := requiredExecutable(t, "VELOX_BUILT_HOST")
	for _, length := range []int{0, 146, 180, 218} {
		t.Run(fmt.Sprintf("profile-%d", length), func(t *testing.T) {
			testNativeStoragePath(t, executable, executable, length)
		})
	}
	if os.Getenv("VELOX_STORAGE_LEGACY_HOST") != "" {
		legacy := requiredExecutable(t, "VELOX_STORAGE_LEGACY_HOST")
		t.Run("legacy-profile", func(t *testing.T) {
			testNativeStoragePath(t, executable, legacy, 0)
		})
	}
}

func testNativeStoragePath(t *testing.T, executable, writer string, length int) {
	allBrowsersExited := false
	root := managedProfileRoot(t, "velox-storage-", func() bool { return allBrowsersExited })
	profile := root
	if len(profile) < length {
		profile = filepath.Join(profile, strings.Repeat("p", length-len(profile)-1))
	}
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	assets := filepath.Join(fixture, "web")
	if err := os.Mkdir(assets, 0o700); err != nil {
		t.Fatal(err)
	}
	const text = "Long-path draft \uD55C\uAE00\r\nSecond line"
	encoded, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"write", "read"} {
		page := `<!doctype html><title>Velox Isolated Storage Test</title><pre id="result">Checking storage</pre><script>
(async () => {
  const expected = ` + string(encoded) + `;
  const mode = "` + mode + `";
  const db = await new Promise((resolve, reject) => {
    const request = indexedDB.open("velox-long-profile-regression", 1);
    request.onupgradeneeded = () => request.result.createObjectStore("drafts");
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
    request.onblocked = () => reject(Error("Database blocked"));
  });
  try {
    await new Promise((resolve, reject) => {
      const tx = db.transaction("drafts", mode === "write" ? "readwrite" : "readonly");
      const store = tx.objectStore("drafts");
      const request = store.get("draft");
      let verified = false;
      request.onsuccess = () => {
        if (mode === "write" && request.result === undefined) {
          store.put(expected, "draft");
          verified = true;
        } else if (mode === "read" && request.result === expected) {
          verified = true;
        } else {
          tx.abort();
        }
      };
      tx.oncomplete = () => verified ? resolve() : reject(Error("Draft not verified"));
      tx.onabort = () => reject(tx.error || Error("Draft mismatch"));
      tx.onerror = () => reject(tx.error);
    });
  } finally { db.close(); }
  document.querySelector("#result").textContent = "Storage " + mode + " verified";
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  await window.__veloxReady("dom-2raf");
})().catch(error => { document.querySelector("#result").textContent = "Storage failed: " + error.name + ": " + error.message; });
</script>`
		if err := os.WriteFile(filepath.Join(assets, mode+".html"), []byte(page), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	configPath := filepath.Join(fixture, "velox.runtime.json")
	host := hostAdapter{
		name: "long-profile-storage", executable: executable,
		arguments: func(string) []string { return []string{"--config", configPath} },
		environment: func(profile string) []string {
			return []string{"VELOX_DATA_DIR=" + profile}
		},
		expectedPhase: "dom-2raf", requireBrowserProcess: true,
	}
	for _, mode := range []string{"write", "read"} {
		host.executable = executable
		if mode == "write" {
			host.executable = writer
		}
		config := map[string]any{
			"runtimeVersion": 1,
			"app":            map[string]any{"id": "dev.velox.storage-path-test", "name": "Velox Isolated Storage Test", "version": "0.1.0"},
			"assets":         map[string]any{"root": "web", "entry": mode + ".html"},
			"window":         map[string]any{"width": 640, "height": 480},
			"security":       map[string]any{"permissions": []string{}},
		}
		data, err := json.Marshal(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
		run, err := runHost(host, profile)
		if err != nil {
			t.Fatalf("native IndexedDB %s failed at profile length %d: %v", mode, len(profile), err)
		}
		if _, err := awaitBrowserExit(run, 10*time.Second); err != nil {
			t.Fatal(err)
		}
		t.Logf("native IndexedDB %s verified; ready=%s host-exit=%s browser-pid=%d profile-length=%d",
			mode, run.Ready, run.Exit, run.BrowserProcessID, len(profile))
	}
	allBrowsersExited = true
	longDatabasePath := false
	if err := filepath.WalkDir(profile, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "MANIFEST-") && strings.Contains(path, ".indexeddb.leveldb") && len(path) >= 260 {
			longDatabasePath = true
			t.Logf("IndexedDB manifest path length=%d", len(path))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if length > 0 && !longDatabasePath {
		t.Fatal("fixture did not exercise an IndexedDB file beyond MAX_PATH")
	}
}
