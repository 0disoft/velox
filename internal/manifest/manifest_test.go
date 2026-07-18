package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDocumentedDefaults(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "web", "index.html"), "ok")
	path := filepath.Join(root, "actutum.json")
	writeTestFile(t, path, `{
  "schemaVersion": 1,
  "app": {"id": "com.example.hello", "name": "Hello", "version": "1.0.0"}
}`)

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Assets.Root != "web" || got.Assets.Entry != "index.html" {
		t.Fatalf("asset defaults = %+v", got.Assets)
	}
	if got.Window.Width != 960 || got.Window.Height != 640 {
		t.Fatalf("window defaults = %+v", got.Window)
	}
	if got.Security.Permissions == nil || len(got.Security.Permissions) != 0 {
		t.Fatalf("permission defaults = %#v", got.Security.Permissions)
	}
}

func TestLoadRejectsInvalidContracts(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{"unknown field", `{"schemaVersion":1,"app":{"id":"com.example.app","name":"App","version":"1"},"extra":true}`, "unknown field"},
		{"future schema", `{"schemaVersion":2,"app":{"id":"com.example.app","name":"App","version":"1"}}`, "unsupported schemaVersion"},
		{"invalid app id", `{"schemaVersion":1,"app":{"id":"Example App","name":"App","version":"1"}}`, "reverse-domain"},
		{"root escape", `{"schemaVersion":1,"app":{"id":"com.example.app","name":"App","version":"1"},"assets":{"root":".."}}`, "stay inside"},
		{"unknown permission", `{"schemaVersion":1,"app":{"id":"com.example.app","name":"App","version":"1"},"security":{"permissions":["shell.execute"]}}`, "unsupported permission"},
		{"duplicate permission", `{"schemaVersion":1,"app":{"id":"com.example.app","name":"App","version":"1"},"security":{"permissions":["app.info","app.info"]}}`, "duplicate permission"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "actutum.json")
			writeTestFile(t, path, test.body)
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Load() error = %v, want containing %q", err, test.message)
			}
		})
	}
}

func writeTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
