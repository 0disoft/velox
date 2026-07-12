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

func TestCreateUsesFullApplicationIDAsCollisionFreeOutputKey(t *testing.T) {
	root := t.TempDir()
	host := []byte("host")
	writePlanFile(t, filepath.Join(root, "release", "velox-host.exe"), string(host))
	writePlanHostMetadata(t, filepath.Join(root, "release"), string(host))
	writePlanFile(t, filepath.Join(root, "web", "index.html"), "ok")
	writePlanFile(t, filepath.Join(root, "velox.json"), `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1"}}`)
	plan, err := Create(Options{ManifestPath: filepath.Join(root, "velox.json"), HostPath: filepath.Join(root, "release", "velox-host.exe")})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Snapshot().ApplicationKey != "com.example.hello" {
		t.Fatalf("application key = %q", plan.Snapshot().ApplicationKey)
	}
}

func TestCreateCanonicalizesRedirectedOutputAncestor(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, filepath.Join(root, "web", "index.html"), "ok")
	writePlanFile(t, filepath.Join(root, "velox-host.exe"), "host")
	writePlanHostMetadata(t, root, "host")
	writePlanFile(t, filepath.Join(root, "velox.json"), `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1"}}`)
	target := t.TempDir()
	redirect := filepath.Join(root, "redirect")
	if err := os.Symlink(target, redirect); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	plan, err := Create(Options{
		ManifestPath: filepath.Join(root, "velox.json"),
		HostPath:     filepath.Join(root, "velox-host.exe"),
		OutputRoot:   filepath.Join(redirect, "dist"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonicalPath(filepath.Join(target, "dist"))
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(plan.Snapshot().OutputRoot, want) {
		t.Fatalf("output root = %q, want %q", plan.Snapshot().OutputRoot, want)
	}
}

func TestCreateCanonicalizesRedirectedHostAncestor(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	writePlanFile(t, filepath.Join(root, "web", "index.html"), "ok")
	writePlanFile(t, filepath.Join(root, "velox.json"), `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1"}}`)
	writePlanFile(t, filepath.Join(target, "velox-host.exe"), "host")
	writePlanHostMetadata(t, target, "host")
	redirect := filepath.Join(root, "release")
	if err := os.Symlink(target, redirect); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	plan, err := Create(Options{
		ManifestPath: filepath.Join(root, "velox.json"),
		HostPath:     filepath.Join(redirect, "velox-host.exe"),
		OutputRoot:   filepath.Join(root, "dist"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonicalPath(filepath.Join(target, "velox-host.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(plan.Snapshot().HostPath, want) {
		t.Fatalf("host path = %q, want %q", plan.Snapshot().HostPath, want)
	}
}

func writePlanHostMetadata(t *testing.T, root, host string) {
	t.Helper()
	digest := sha256.Sum256([]byte(host))
	body := fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.4.0-dev","target":"windows-x64","contracts":{"host":1,"runtime":1},"host":{"file":"velox-host.exe","bytes":%d,"sha256":"%x"}}`, len(host), digest)
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
