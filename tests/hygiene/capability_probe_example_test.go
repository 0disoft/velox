package hygiene_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapabilityProbeStaysBrowserOwnedAndDependencyFree(t *testing.T) {
	root := repositoryRoot(t)
	manifestPath := filepath.Join(root, "examples", "capability-probe", "velox.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
		Security struct {
			Permissions []string `json:"permissions"`
		} `json:"security"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.App.ID != "dev.velox.capabilityprobe" {
		t.Fatalf("unexpected capability probe ID %q", manifest.App.ID)
	}
	if len(manifest.Security.Permissions) != 1 || manifest.Security.Permissions[0] != "app.info" {
		t.Fatalf("capability probe widened native permissions: %v", manifest.Security.Permissions)
	}

	index := readCapabilityProbeText(t, filepath.Join(root, "examples", "capability-probe", "web", "index.html"))
	for _, marker := range []string{
		"connect-src 'none'",
		"aria-live=\"polite\"",
		"role=\"button\"",
		"app.js",
	} {
		if !strings.Contains(index, marker) {
			t.Fatalf("capability probe index missing %q", marker)
		}
	}

	script := readCapabilityProbeText(t, filepath.Join(root, "examples", "capability-probe", "web", "app.js"))
	for _, marker := range []string{
		"window.showOpenFilePicker",
		"window.showSaveFilePicker",
		"navigator.clipboard.writeText",
		"indexedDB.open",
		"window.velox.invoke(\"app.getInfo\"",
		"window.__veloxReady(\"dom-2raf\")",
	} {
		if !strings.Contains(script, marker) {
			t.Fatalf("capability probe script missing %q", marker)
		}
	}
	for _, forbidden := range []string{"shell.exec", "process.exec", "filesystem.read", "http://", "https://"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("capability probe script contains forbidden native or network surface %q", forbidden)
		}
	}
}

func readCapabilityProbeText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
