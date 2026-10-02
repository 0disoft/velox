package runtimeconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/manifest"
)

func TestWindowRememberStateManifestRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		value := manifest.Resolved{Manifest: manifest.Manifest{App: manifest.App{ID: "dev.velox.state-test", Name: "State test", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"}, Window: manifest.Window{Width: 800, Height: 600, RememberState: enabled}}}
		cfg := FromManifest(value, "web")
		body, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(body)
		if err != nil || parsed.Window.RememberState != enabled {
			t.Fatalf("round trip = %+v, %v", parsed.Window, err)
		}
		if strings.Contains(string(body), `"rememberState"`) != enabled {
			t.Fatalf("optional field output = %s", body)
		}
	}
}

func TestAppSingleInstanceManifestRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		value := manifest.Resolved{Manifest: manifest.Manifest{App: manifest.App{ID: "dev.velox.instance-test", Name: "Test", Version: "1", SingleInstance: enabled},
			Assets: manifest.Assets{Entry: "index.html"}, Window: manifest.Window{Width: 800, Height: 600}}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(body)
		if err != nil || parsed.App.SingleInstance != enabled || strings.Contains(string(body), `"singleInstance"`) != enabled {
			t.Fatalf("round trip=%+v, %s, %v", parsed.App, body, err)
		}
		if enabled {
			invalid := strings.Replace(string(body), `"singleInstance":true`, `"singleInstance":"true"`, 1)
			if _, err := Parse([]byte(invalid)); err == nil {
				t.Fatal("accepted non-boolean singleInstance")
			}
		}
	}
}

func TestWindowTrayManifestRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		value := manifest.Resolved{Manifest: manifest.Manifest{App: manifest.App{ID: "dev.velox.tray-test", Name: "Tray", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"}, Window: manifest.Window{Width: 800, Height: 600, Tray: enabled}}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(body)
		if err != nil || parsed.Window.Tray != enabled || strings.Contains(string(body), `"tray"`) != enabled {
			t.Fatalf("round trip=%+v, %s, %v", parsed.Window, body, err)
		}
		if enabled {
			invalid := strings.Replace(string(body), `"tray":true`, `"tray":"true"`, 1)
			if _, err := Parse([]byte(invalid)); err == nil {
				t.Fatal("accepted non-boolean tray")
			}
		}
	}
}

func TestLoad(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	if err := os.Mkdir(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "velox.runtime.json")
	config := `{
  "runtimeVersion": 1,
  "app": {"id": "dev.velox.hello", "name": "Hello", "version": "1.0.0"},
  "assets": {"root": "web", "entry": "index.html"},
  "window": {"width": 640, "height": 480},
  "security": {"permissions": []}
}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.EntryPath != filepath.Join(web, "index.html") {
		t.Fatalf("EntryPath = %q", got.EntryPath)
	}
}

func TestLoadRejectsUnsafeOrUnknownInput(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		message string
	}{
		{
			name:    "unsupported version",
			config:  `{"runtimeVersion":2,"app":{"id":"dev.test.x","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`,
			message: "unsupported runtimeVersion",
		},
		{
			name:    "invalid app id",
			config:  `{"runtimeVersion":1,"app":{"id":"../../escaped-profile","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`,
			message: "reverse-domain",
		},
		{
			name:    "root escape",
			config:  `{"runtimeVersion":1,"app":{"id":"dev.test.x","name":"x","version":"1"},"assets":{"root":"..","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`,
			message: "path must stay inside",
		},
		{
			name:    "unknown field",
			config:  `{"runtimeVersion":1,"app":{"id":"dev.test.x","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]},"surprise":true}`,
			message: "unknown field",
		},
		{
			name:    "multiple values",
			config:  `{"runtimeVersion":1,"app":{"id":"dev.test.x","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}} {}`,
			message: "multiple JSON values",
		},
		{
			name:    "missing permissions",
			config:  `{"runtimeVersion":1,"app":{"id":"dev.test.x","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{}}`,
			message: "security.permissions is required",
		},
		{
			name:    "unknown permission",
			config:  `{"runtimeVersion":1,"app":{"id":"dev.test.x","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":["shell.execute"]}}`,
			message: "unsupported permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "velox.runtime.json")
			if err := os.WriteFile(path, []byte(tt.config), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.message)
			}
		})
	}
}

func TestLoadRejectsLinkedAssetBoundary(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "index.html"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "web")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	config := `{"runtimeVersion":1,"app":{"id":"dev.test.linked","name":"x","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`
	path := filepath.Join(root, "velox.runtime.json")
	if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "link or reparse point") {
		t.Fatalf("Load() error = %v, want linked asset rejection", err)
	}
}

func TestContainedPathRejectsWindowsDriveRelativePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows drive-relative paths are platform-specific")
	}
	if _, err := containedPath(t.TempDir(), `C:outside`); err == nil || !strings.Contains(err.Error(), "absolute paths") {
		t.Fatalf("containedPath() error = %v, want a Windows volume rejection", err)
	}
}
