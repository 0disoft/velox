package startup_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise the shipped bridge in an actual runtime without opening dialogs or
// writing user files. Valid input must reach host permission checks; malformed
// UTF-16 must fail in JavaScript before TextEncoder can replace its surrogates.
func TestBuiltHostSaveBridgeCompatibility(t *testing.T) {
	version := os.Getenv("VELOX_SAVE_BRIDGE_BROWSER_VERSION")
	if version == "" {
		t.Skip("opt-in actual runtime save-bridge compatibility witness")
	}
	major, err := strconv.Atoi(strings.Split(version, ".")[0])
	if err != nil {
		t.Fatal(err)
	}
	executable := requiredExecutable(t, "VELOX_BUILT_HOST")
	fixture := t.TempDir()
	assets := filepath.Join(fixture, "web")
	if err := os.Mkdir(assets, 0o700); err != nil {
		t.Fatal(err)
	}
	expected, _ := json.Marshal(version)
	page := fmt.Sprintf(`<!doctype html><title>Save bridge compatibility</title><p id="result">Checking</p><script>
(async () => {
  const actual = (navigator.userAgent.match(/Edg\/([0-9.]+)/) || [])[1];
  if (actual !== %s) throw Error("Unexpected runtime: " + actual);
  if ((typeof String.prototype.isWellFormed === "function") !== %t) throw Error("Unexpected builtin availability");
  async function expectError(operation, code) {
    try { await operation(); } catch (error) {
      if (error.code === code) return;
      throw error;
    }
    throw Error("Expected rejection: " + code);
  }
  const text = "a".repeat(4095) + "\uD83D\uDE00\uD55C\uAE00\r\n";
  await expectError(() => window.velox.saveText(text), "PERMISSION_DENIED");
  await expectError(() => window.velox.saveTextAs(text), "PERMISSION_DENIED");
  await expectError(() => window.velox.saveTextTo(text, 7), "PERMISSION_DENIED");
  for (const invalid of ["\uD800", "\uDFFF", "\uD800x", "x\uDFFF", "a\0b"]) {
    await expectError(() => window.velox.saveText(invalid), "INVALID_PARAMS");
  }
  document.querySelector("#result").textContent = "Save bridge verified";
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  await window.__veloxReady("dom-2raf");
})().catch(error => {
  document.querySelector("#result").textContent = error.name + ": " + error.message;
  // Use the other supported marker so the parent fails promptly on errors.
  window.__veloxReady("security-ok");
});
</script>`, expected, major >= 111)
	if err := os.WriteFile(filepath.Join(assets, "index.html"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{"runtimeVersion": 1,
		"app":      map[string]any{"id": "dev.velox.save-bridge-test", "name": "Save Bridge Test", "version": "1"},
		"assets":   map[string]any{"root": "web", "entry": "index.html"},
		"window":   map[string]any{"width": 640, "height": 480},
		"security": map[string]any{"permissions": []string{}}}
	body, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(fixture, "velox.runtime.json")
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	cleanupSafe := false
	profile := managedProfileRoot(t, "velox-save-bridge-", func() bool { return cleanupSafe })
	host := hostAdapter{name: "save-bridge", executable: executable,
		arguments:     func(string) []string { return []string{"--config", configPath} },
		environment:   func(profile string) []string { return []string{"VELOX_DATA_DIR=" + profile} },
		expectedPhase: "dom-2raf", requireBrowserProcess: true}
	run, err := runHost(host, profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := awaitBrowserExit(run, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	cleanupSafe = true
	t.Logf("actual runtime=%s builtin-present=%t: 3 save helpers reached permission checks; 5 malformed strings rejected", version, major >= 111)
}
