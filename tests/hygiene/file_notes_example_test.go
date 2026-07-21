package hygiene_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileNotesUsesOnlyBrowserOwnedFileAccess(t *testing.T) {
	root := repositoryRoot(t)
	manifestData, err := os.ReadFile(filepath.Join(root, "examples", "file-notes", "velox.json"))
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
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.App.ID != "dev.velox.filenotes" || len(manifest.Security.Permissions) != 0 {
		t.Fatalf("file-notes widened the native boundary: id=%q permissions=%v", manifest.App.ID, manifest.Security.Permissions)
	}

	files := map[string][]string{
		"index.html": {"connect-src 'none'", "<textarea", "<dialog", "aria-live=\"polite\""},
		"app.js":     {"showOpenFilePicker", "showSaveFilePicker", "maximumFileBytes", "beforeunload", "restore().finally(reportReady)", "window.__veloxReady(\"dom-2raf\")"},
		"model.js":   {"savedText", "isDirty", "Object.freeze"},
		"storage.js": {"indexedDB.open", "DataCloneError", "Object.freeze"},
		"style.css":  {"minmax(0, 1fr)", "overflow-wrap: anywhere", "@media (max-width: 650px)", ":focus-visible"},
	}
	for name, markers := range files {
		data, err := os.ReadFile(filepath.Join(root, "examples", "file-notes", "web", name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		for _, marker := range markers {
			if !strings.Contains(body, marker) {
				t.Errorf("file-notes %s lacks %q", name, marker)
			}
		}
		for _, forbidden := range []string{"window.velox.invoke", "shell.exec", "process.exec", "filesystem.", "http://", "https://"} {
			if strings.Contains(body, forbidden) {
				t.Errorf("file-notes %s contains forbidden surface %q", name, forbidden)
			}
		}
	}
}
