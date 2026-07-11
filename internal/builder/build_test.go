package builder

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/buildplan"
)

func TestBuildIsDeterministicAndKeepsHostUnchanged(t *testing.T) {
	root, manifestPath, hostPath := fixture(t)
	plan, err := buildplan.Create(buildplan.Options{
		ManifestPath: manifestPath,
		HostPath:     hostPath,
		OutputRoot:   filepath.Join(root, "dist"),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	firstArchive, err := os.ReadFile(first.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	secondArchive, err := os.ReadFile(second.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if first.ArchiveSHA256 != second.ArchiveSHA256 || !bytes.Equal(firstArchive, secondArchive) {
		t.Fatalf("archive is not deterministic: %s != %s", first.ArchiveSHA256, second.ArchiveSHA256)
	}
	sourceHost, _ := os.ReadFile(hostPath)
	copiedHost, _ := os.ReadFile(filepath.Join(second.DirectoryPath, "hello.exe"))
	if !bytes.Equal(sourceHost, copiedHost) {
		t.Fatal("packaged host bytes changed")
	}

	reader, err := zip.OpenReader(second.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != second.Report.Outputs.PortableFiles {
		t.Fatalf("archive files = %d, want %d", len(reader.File), second.Report.Outputs.PortableFiles)
	}
	for _, file := range reader.File {
		if file.Modified.Year() != 1980 {
			t.Fatalf("archive timestamp for %s = %s", file.Name, file.Modified)
		}
	}
}

func TestBuildPreservesPreviousOutputWhenStagingIsOccupied(t *testing.T) {
	root, manifestPath, hostPath := fixture(t)
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: hostPath, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(result.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(plan.Snapshot().OutputRoot, ".hello.staging"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(plan); err == nil {
		t.Fatal("Build() succeeded with occupied staging")
	}
	after, err := os.ReadFile(result.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("previous archive changed after failed build")
	}
}

func TestBuildReportIsStableAndContainsNoAbsolutePaths(t *testing.T) {
	root, manifestPath, hostPath := fixture(t)
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: hostPath, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(result.DirectoryPath, "build-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(root)) || !json.Valid(data) {
		t.Fatalf("unsafe build report: %s", data)
	}
}

func TestBuildRejectsSourceChangesAfterPlanning(t *testing.T) {
	root, manifestPath, hostPath := fixture(t)
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: hostPath, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(root, "web", "app.js"), []byte("changed after planning\n"))
	if _, err := Build(plan); err == nil {
		t.Fatal("Build() accepted an asset changed after planning")
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "hello.zip")); !os.IsNotExist(err) {
		t.Fatalf("failed build promoted an archive: %v", err)
	}
}

func fixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	manifestPath := filepath.Join(root, "velox.json")
	hostPath := filepath.Join(root, "release", "velox-host.exe")
	host := []byte("prebuilt-host\x00bytes")
	writeFixture(t, hostPath, host)
	writeFixture(t, filepath.Join(filepath.Dir(hostPath), "velox-host.json"), hostMetadata(host))
	writeFixture(t, filepath.Join(root, "web", "index.html"), []byte("<!doctype html><title>Hello</title>"))
	writeFixture(t, filepath.Join(root, "web", "app.js"), []byte("console.log('hello')\n"))
	writeFixture(t, manifestPath, []byte(`{
  "schemaVersion": 1,
  "app": {"id": "com.example.hello", "name": "Hello", "version": "1.2.3"},
  "assets": {"root": "web", "entry": "index.html"},
  "window": {"width": 720, "height": 480},
  "security": {"permissions": ["app.info"]}
}`))
	return root, manifestPath, hostPath
}

func hostMetadata(host []byte) []byte {
	digest := sha256.Sum256(host)
	return []byte(fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.3.0-dev","target":"windows-x64","contracts":{"host":1,"runtime":1},"host":{"file":"velox-host.exe","bytes":%d,"sha256":"%x"}}`, len(host), digest))
}

func writeFixture(t *testing.T, path string, value []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, value, 0o644); err != nil {
		t.Fatal(err)
	}
}
