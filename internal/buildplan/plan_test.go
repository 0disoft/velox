package buildplan

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotCannotMutatePlan(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, filepath.Join(root, "web", "index.html"), "ok")
	writePlanFile(t, filepath.Join(root, "velox-host.exe"), "host")
	writePlanHostMetadata(t, root, "host")
	writePlanFile(t, filepath.Join(root, "velox.json"), `{
  "schemaVersion": 1,
  "app": {"id": "com.example.hello", "name": "Hello", "version": "1"},
  "security": {"permissions": ["app.info"]}
}`)
	plan, err := Create(Options{
		ManifestPath: filepath.Join(root, "velox.json"),
		HostPath:     filepath.Join(root, "velox-host.exe"),
		OutputRoot:   filepath.Join(root, "dist"),
	})
	if err != nil {
		t.Fatal(err)
	}
	first := plan.Snapshot()
	first.Manifest.Security.Permissions[0] = "window.basic"
	first.Assets.Files[0].RelativePath = "changed"
	second := plan.Snapshot()
	if second.Manifest.Security.Permissions[0] != "app.info" || second.Assets.Files[0].RelativePath != "index.html" {
		t.Fatalf("snapshot mutated plan: %+v", second)
	}
}

func TestCreateRejectsOutputContainingAssets(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, filepath.Join(root, "web", "index.html"), "ok")
	writePlanFile(t, filepath.Join(root, "velox-host.exe"), "host")
	writePlanHostMetadata(t, root, "host")
	writePlanFile(t, filepath.Join(root, "velox.json"), `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1"}}`)
	_, err := Create(Options{
		ManifestPath: filepath.Join(root, "velox.json"),
		HostPath:     filepath.Join(root, "velox-host.exe"),
		OutputRoot:   root,
	})
	if err == nil {
		t.Fatal("Create() accepted output root containing assets")
	}
}

func TestCreateRejectsRedirectedOutputRoot(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, filepath.Join(root, "web", "index.html"), "ok")
	writePlanFile(t, filepath.Join(root, "velox-host.exe"), "host")
	writePlanHostMetadata(t, root, "host")
	writePlanFile(t, filepath.Join(root, "velox.json"), `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1"}}`)
	redirect := filepath.Join(root, "redirect")
	if err := os.Symlink(t.TempDir(), redirect); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, err := Create(Options{
		ManifestPath: filepath.Join(root, "velox.json"),
		HostPath:     filepath.Join(root, "velox-host.exe"),
		OutputRoot:   filepath.Join(redirect, "dist"),
	})
	if err == nil {
		t.Fatal("Create() accepted redirected output root")
	}
}

func writePlanHostMetadata(t *testing.T, root, host string) {
	t.Helper()
	digest := sha256.Sum256([]byte(host))
	body := fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.3.0-dev","target":"windows-x64","contracts":{"host":1,"runtime":1},"host":{"file":"velox-host.exe","bytes":%d,"sha256":"%x"}}`, len(host), digest)
	writePlanFile(t, filepath.Join(root, "velox-host.json"), body)
}

func writePlanFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
