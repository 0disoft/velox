package runtimeconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
  "app": {"id": "dev.velox.hello", "name": "Hello"},
  "assets": {"root": "web", "entry": "index.html"},
  "window": {"width": 640, "height": 480}
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
			config:  `{"runtimeVersion":2,"app":{"id":"x","name":"x"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480}}`,
			message: "unsupported runtimeVersion",
		},
		{
			name:    "root escape",
			config:  `{"runtimeVersion":1,"app":{"id":"x","name":"x"},"assets":{"root":"..","entry":"index.html"},"window":{"width":640,"height":480}}`,
			message: "path must stay inside",
		},
		{
			name:    "unknown field",
			config:  `{"runtimeVersion":1,"app":{"id":"x","name":"x"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"surprise":true}`,
			message: "unknown field",
		},
		{
			name:    "multiple values",
			config:  `{"runtimeVersion":1,"app":{"id":"x","name":"x"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480}} {}`,
			message: "multiple JSON values",
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
