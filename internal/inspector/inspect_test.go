package inspector

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/builder"
	"github.com/0disoft/velox/internal/buildplan"
)

func TestInspectValidatesDirectoryAndZIP(t *testing.T) {
	build := buildFixture(t)
	for _, test := range []struct {
		path string
		kind string
	}{{build.DirectoryPath, "directory"}, {build.ArchivePath, "zip"}} {
		result, err := Inspect(test.path)
		if err != nil {
			t.Fatalf("Inspect(%s): %v", test.kind, err)
		}
		if result.Kind != test.kind || result.ReleaseVersion != "0.4.1-dev" || result.App.ID != "com.example.inspect" || result.PortableFiles != 4 {
			t.Fatalf("unexpected %s result: %+v", test.kind, result)
		}
	}
}

func TestInspectRejectsTamperedHost(t *testing.T) {
	build := buildFixture(t)
	if err := os.WriteFile(filepath.Join(build.DirectoryPath, "inspect.exe"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(build.DirectoryPath); err == nil {
		t.Fatal("Inspect() accepted a tampered host")
	}
}

func TestInspectRejectsUnsafeZIPPaths(t *testing.T) {
	for _, names := range [][]string{{"app/../escape"}, {"app/File.txt", "app/file.txt"}, {"one/a", "two/b"}} {
		archivePath := filepath.Join(t.TempDir(), "unsafe.zip")
		file, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		for _, name := range names {
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = entry.Write([]byte("x"))
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := Inspect(archivePath); err == nil {
			t.Fatalf("Inspect() accepted unsafe entries: %v", names)
		}
	}
}

func TestValidateArchiveBudgetRejectsUnsafeEntries(t *testing.T) {
	tests := []struct {
		name string
		file *zip.File
	}{
		{
			name: "oversized entry",
			file: &zip.File{FileHeader: zip.FileHeader{Name: "app/large.bin", UncompressedSize64: maxArchiveEntryBytes + 1, CompressedSize64: maxArchiveEntryBytes + 1}},
		},
		{
			name: "expansion ratio",
			file: &zip.File{FileHeader: zip.FileHeader{Name: "app/bomb.bin", UncompressedSize64: maxArchiveExpandRatio + 1, CompressedSize64: 1}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateArchiveBudget([]*zip.File{test.file}); err == nil {
				t.Fatal("validateArchiveBudget() accepted an unsafe entry")
			}
		})
	}
}

func buildFixture(t *testing.T) builder.Result {
	t.Helper()
	root := t.TempDir()
	host := []byte("host")
	hostPath := filepath.Join(root, "release", "velox-host.exe")
	writeInspectFile(t, hostPath, host)
	digest := sha256.Sum256(host)
	metadata := fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.4.1-dev","target":"windows-x64","contracts":{"host":1,"runtime":1},"host":{"file":"velox-host.exe","bytes":%d,"sha256":"%x"}}`, len(host), digest)
	writeInspectFile(t, filepath.Join(filepath.Dir(hostPath), "velox-host.json"), []byte(metadata))
	writeInspectFile(t, filepath.Join(root, "web", "index.html"), []byte("<title>Inspect</title>"))
	manifest := `{"schemaVersion":1,"app":{"id":"com.example.inspect","name":"Inspect","version":"1.0.0"}}`
	manifestPath := filepath.Join(root, "velox.json")
	writeInspectFile(t, manifestPath, []byte(manifest))
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: hostPath, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := builder.Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writeInspectFile(t *testing.T, path string, value []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, value, 0o644); err != nil {
		t.Fatal(err)
	}
}
