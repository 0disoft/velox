package builder

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
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
	copiedHost, _ := os.ReadFile(filepath.Join(second.DirectoryPath, "com.example.hello.exe"))
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
	if err := os.Mkdir(filepath.Join(plan.Snapshot().OutputRoot, ".com.example.hello.staging"), 0o755); err != nil {
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
	if _, err := os.Stat(filepath.Join(root, "dist", "com.example.hello.zip")); !os.IsNotExist(err) {
		t.Fatalf("failed build promoted an archive: %v", err)
	}
}

func TestBuildRejectsAssetsAddedAfterPlanning(t *testing.T) {
	root, manifestPath, hostPath := fixture(t)
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: hostPath, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(root, "web", "late.js"), []byte("late\n"))
	if _, err := Build(plan); err == nil {
		t.Fatal("Build() silently omitted an asset added after planning")
	}
}

func TestBuildRejectsInvalidExistingArchiveBeforePromotion(t *testing.T) {
	root, manifestPath, hostPath := fixture(t)
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: manifestPath, HostPath: hostPath, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(plan.Snapshot().ArchivePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(plan); err == nil {
		t.Fatal("Build() accepted a directory at the final archive path")
	}
	if _, err := os.Stat(plan.Snapshot().AppDirectory); !os.IsNotExist(err) {
		t.Fatalf("failed build promoted the app directory: %v", err)
	}
}

func TestPromoteReportsPrimaryAndRollbackFailures(t *testing.T) {
	root := t.TempDir()
	plan := buildplan.Snapshot{
		AppDirectory: filepath.Join(root, "app"),
		ArchivePath:  filepath.Join(root, "app.zip"),
	}
	if err := os.Mkdir(plan.AppDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, plan.ArchivePath, []byte("previous"))

	promoteErr := errors.New("promote failed")
	removeErr := errors.New("remove failed")
	restoreErr := errors.New("restore failed")
	stageDirectory := filepath.Join(root, "stage-app")
	stageArchive := filepath.Join(root, "stage-app.zip")
	err := promoteWithOperations(plan, stageDirectory, stageArchive, promotionOperations{
		rename: func(source, destination string) error {
			switch source {
			case stageDirectory:
				return promoteErr
			case plan.AppDirectory + ".previous":
				return restoreErr
			default:
				return nil
			}
		},
		removeAll: func(path string) error {
			if path == plan.AppDirectory {
				return removeErr
			}
			return nil
		},
		remove: func(string) error { return nil },
	})
	for _, expected := range []error{promoteErr, removeErr, restoreErr} {
		if !errors.Is(err, expected) {
			t.Fatalf("promote error %v does not retain %v", err, expected)
		}
	}
}

func TestPromoteReportsArchiveBackupAndDirectoryRestoreFailures(t *testing.T) {
	root := t.TempDir()
	plan := buildplan.Snapshot{
		AppDirectory: filepath.Join(root, "app"),
		ArchivePath:  filepath.Join(root, "app.zip"),
	}
	if err := os.Mkdir(plan.AppDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, plan.ArchivePath, []byte("previous"))

	backupErr := errors.New("archive backup failed")
	restoreErr := errors.New("directory restore failed")
	err := promoteWithOperations(plan, filepath.Join(root, "stage-app"), filepath.Join(root, "stage-app.zip"), promotionOperations{
		rename: func(source, destination string) error {
			switch source {
			case plan.ArchivePath:
				return backupErr
			case plan.AppDirectory + ".previous":
				return restoreErr
			default:
				return nil
			}
		},
		removeAll: func(string) error { return nil },
		remove:    func(string) error { return nil },
	})
	for _, expected := range []error{backupErr, restoreErr} {
		if !errors.Is(err, expected) {
			t.Fatalf("backup error %v does not retain %v", err, expected)
		}
	}
}

func TestPromoteReportsPreviousOutputCleanupFailure(t *testing.T) {
	root := t.TempDir()
	plan := buildplan.Snapshot{
		AppDirectory: filepath.Join(root, "app"),
		ArchivePath:  filepath.Join(root, "app.zip"),
	}
	if err := os.Mkdir(plan.AppDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, plan.ArchivePath, []byte("previous"))

	cleanupErr := errors.New("cleanup failed")
	err := promoteWithOperations(plan, filepath.Join(root, "stage-app"), filepath.Join(root, "stage-app.zip"), promotionOperations{
		rename: func(string, string) error { return nil },
		removeAll: func(path string) error {
			if path == plan.AppDirectory+".previous" {
				return cleanupErr
			}
			return nil
		},
		remove: func(string) error { return nil },
	})
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("promote cleanup error = %v, want %v", err, cleanupErr)
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
	return []byte(fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.5.10-alpha.2","target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"velox-host.exe","bytes":%d,"sha256":"%x"}}`, len(host), digest))
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
