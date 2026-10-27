package startup_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in: an actual previous host is required. Keep one directory and committed
// IndexedDB draft through all three launches; release helpers delete profiles.
func TestBuiltHostUpgradeHandoff(t *testing.T) {
	previousPath := os.Getenv("VELOX_UPGRADE_PREVIOUS_HOST")
	if previousPath == "" {
		t.Skip("upgrade handoff requires an explicit previous host")
	}
	nextPath := requiredExecutable(t, "VELOX_BUILT_HOST")
	for _, path := range []string{previousPath, nextPath} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("host=%s sha256=%x", path, sha256.Sum256(body))
	}

	cleanupSafe := false
	profile := managedProfileRoot(t, "velox-upgrade-", func() bool { return cleanupSafe })
	original, err := os.Stat(profile)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(profile, "upgrade-marker.txt")
	if err := os.WriteFile(marker, []byte("original profile"), 0o600); err != nil {
		t.Fatal(err)
	}
	var runs []hostRun
	unobservedStart := false
	defer func() {
		cleanupSafe = !unobservedStart
		for index, run := range runs {
			exitedAt, err := awaitBrowserExitAt(run, 10*time.Second)
			if err != nil {
				cleanupSafe = false
				t.Error(err)
				continue
			}
			t.Logf("launch=%d ready=%s browser-exit-after-host=%s", index, run.Ready, exitedAt.Sub(run.HostExitedAt))
			if index == 0 && len(runs) > 1 {
				t.Logf("new-host-ready-after-old-browser-exit=%s", runs[1].ReadyAt.Sub(exitedAt))
				t.Logf("handoff-ready-after-old-host-exit=%s", runs[1].ReadyAt.Sub(run.HostExitedAt))
			}
		}
	}()

	fixture := t.TempDir()
	assets := filepath.Join(fixture, "web")
	if err := os.Mkdir(assets, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(fixture, "velox.runtime.json")
	// Launch immediately after each host exits. Runtime handoff belongs to the
	// product; a fixture-side browser-exit wait would hide regressions there.
	for index, executable := range []string{previousPath, nextPath, nextPath} {
		mode, version := "read", os.Getenv("VELOX_UPGRADE_NEXT_VERSION")
		if index == 0 {
			mode, version = "write", os.Getenv("VELOX_UPGRADE_PREVIOUS_VERSION")
		}
		t.Logf("launch=%d mode=%s required-runtime=%s", index, mode, version)
		// The witness reports readiness only after a committed write or an exact
		// Korean/CRLF draft read. It never closes or kills the browser itself.
		page := crashStoragePage(mode)
		expected, _ := json.Marshal(version)
		if version != "" {
			page = strings.Replace(page, "const mode =", "const actual = (navigator.userAgent.match(/Edg\\/([0-9.]+)/) || [])[1];\nif (actual !== "+string(expected)+") throw Error('Unexpected-runtime-' + actual);\nconst mode =", 1)
		}
		// Distinct URLs keep a cached write page from masking a failed read.
		entry := fmt.Sprintf("phase-%d.html", index)
		if err := os.WriteFile(filepath.Join(assets, entry), []byte(page), 0o600); err != nil {
			t.Fatal(err)
		}
		config := map[string]any{"runtimeVersion": 1,
			"app":      map[string]any{"id": "dev.velox.upgrade-handoff", "name": "Velox Upgrade Handoff", "version": "1"},
			"assets":   map[string]any{"root": "web", "entry": entry},
			"window":   map[string]any{"width": 640, "height": 480},
			"security": map[string]any{"permissions": []string{}}}
		body, err := json.Marshal(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, body, 0o600); err != nil {
			t.Fatal(err)
		}
		host := hostAdapter{name: fmt.Sprintf("upgrade-%d", index), executable: executable,
			arguments: func(string) []string { return []string{"--config", configPath} },
			environment: func(profile string) []string {
				env := []string{"VELOX_DATA_DIR=" + profile}
				if index == 0 && os.Getenv("VELOX_UPGRADE_PREVIOUS_BROWSER_DIR") != "" {
					env = append(env, "VELOX_BENCH_WEBVIEW2_BROWSER_DIR="+os.Getenv("VELOX_UPGRADE_PREVIOUS_BROWSER_DIR"))
				}
				return env
			},
			expectedPhase: "dom-2raf", requireBrowserProcess: true}
		run, err := runHost(host, profile)
		if err != nil {
			// A failed startup may still have created a browser that was not
			// observable through the ready pipe. Preserve its profile on failure.
			unobservedStart = true
			t.Fatal(err)
		}
		runs = append(runs, run)
		assertStartupTimeline(t, run.Timeline)
		current, err := os.Stat(profile)
		if err != nil || !os.SameFile(original, current) {
			t.Fatalf("profile identity changed: %v", err)
		}
		if body, err := os.ReadFile(marker); err != nil || string(body) != "original profile" {
			t.Fatalf("profile marker changed: %q %v", body, err)
		}
	}
}
