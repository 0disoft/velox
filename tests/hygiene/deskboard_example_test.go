package hygiene_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskboardExampleKeepsTheStaticApplicationBoundary(t *testing.T) {
	root := repositoryRoot(t)
	manifestData, err := os.ReadFile(filepath.Join(root, "examples", "deskboard", "velox.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SchemaVersion int `json:"schemaVersion"`
		App           struct {
			ID string `json:"id"`
		} `json:"app"`
		Security struct {
			Permissions []string `json:"permissions"`
		} `json:"security"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || manifest.App.ID != "dev.velox.deskboard" {
		t.Fatalf("Deskboard manifest identity drifted: %+v", manifest)
	}
	if strings.Join(manifest.Security.Permissions, ",") != "app.info,window.basic" {
		t.Fatalf("Deskboard permissions = %v", manifest.Security.Permissions)
	}

	markers := map[string][]string{
		"index.html": {
			"connect-src 'none'",
			"<form class=\"task-form\"",
			"<dialog id=\"app-dialog\"",
			"role=\"status\"",
		},
		"app.js": {
			"dev.velox.deskboard.tasks.v1",
			"window.velox.invoke",
			"app.getInfo",
			"window.getState",
			"window.__veloxReady(\"dom-2raf\")",
		},
		"model.js": {
			"schemaVersion = 1",
			"normalizeState",
			"selectTasks",
			"Object.freeze",
		},
		"style.css": {
			"grid-template-columns: 232px minmax(0, 1fr)",
			"overflow-wrap: anywhere",
			"@media (max-width: 520px)",
			":focus-visible",
		},
	}
	for name, required := range markers {
		data, err := os.ReadFile(filepath.Join(root, "examples", "deskboard", "web", name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		for _, marker := range required {
			if !strings.Contains(body, marker) {
				t.Errorf("Deskboard %s lacks %q", name, marker)
			}
		}
		if strings.Contains(body, "https://") || strings.Contains(body, "http://") {
			t.Errorf("Deskboard %s contains a remote URL", name)
		}
	}
}
