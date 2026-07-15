package hostmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndValidateArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox-host.json")
	writeMetadata(t, path, `{
  "schemaVersion": "velox.host/v1",
  "releaseVersion": "0.5.1-dev",
  "target": "windows-x64",
  "contracts": {"host": 1, "runtime": 1, "ipc": 1},
  "host": {"file": "velox-host.exe", "bytes": 4, "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
}`)
	metadata, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := metadata.ValidateArtifact(filepath.Join(filepath.Dir(path), "velox-host.exe"), "windows-x64", "0.5.1-dev", 1, 1, 4, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsMalformedMetadata(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"unknown field", `{"schemaVersion":"velox.host/v1","releaseVersion":"x","target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"velox-host.exe","bytes":1,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"extra":true}`},
		{"path file", `{"schemaVersion":"velox.host/v1","releaseVersion":"x","target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"../host.exe","bytes":1,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`},
		{"uppercase digest", `{"schemaVersion":"velox.host/v1","releaseVersion":"x","target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"velox-host.exe","bytes":1,"sha256":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "velox-host.json")
			writeMetadata(t, path, test.body)
			if _, err := Load(path); err == nil {
				t.Fatal("Load() succeeded")
			}
		})
	}
}

func writeMetadata(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
