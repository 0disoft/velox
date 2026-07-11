package releasebundle

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/hostmeta"
)

func TestBuildCreatesDeterministicSelfDescribingBundle(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	cliPath := filepath.Join(root, "input", "velox.exe")
	hostPath := filepath.Join(root, "input", "velox-host.exe")
	writeReleaseFile(t, cliPath, []byte("cli-binary"))
	writeReleaseFile(t, hostPath, []byte("host-binary"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "schema", "velox-v1.schema.json"), []byte("{}\n"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "schema", "host-metadata-v1.schema.json"), []byte("{}\n"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "THIRD_PARTY_NOTICES.md"), []byte("notices\n"))

	first, err := Build(Options{CLIPath: cliPath, HostPath: hostPath, SourceRoot: sourceRoot, OutputRoot: filepath.Join(root, "first")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(Options{CLIPath: cliPath, HostPath: hostPath, SourceRoot: sourceRoot, OutputRoot: filepath.Join(root, "second")})
	if err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := os.ReadFile(first.Archive)
	secondBytes, _ := os.ReadFile(second.Archive)
	if first.ArchiveSHA256 != second.ArchiveSHA256 || !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("release bundle is not deterministic: %s != %s", first.ArchiveSHA256, second.ArchiveSHA256)
	}

	metadata, err := hostmeta.Load(filepath.Join(first.Directory, "velox-host.json"))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Host.File != "velox-host.exe" || metadata.Host.Bytes != int64(len("host-binary")) {
		t.Fatalf("unexpected host metadata: %+v", metadata)
	}
	data, err := os.ReadFile(filepath.Join(first.Directory, "release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != SchemaVersion || len(manifest.Artifacts) != 6 {
		t.Fatalf("unexpected release manifest: %+v", manifest)
	}
	for index := 1; index < len(manifest.Artifacts); index++ {
		if manifest.Artifacts[index-1].File >= manifest.Artifacts[index].File {
			t.Fatalf("release artifacts are not sorted: %+v", manifest.Artifacts)
		}
	}
}

func TestBuildReplacesExistingReleaseAtomically(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	cliPath := filepath.Join(root, "velox.exe")
	hostPath := filepath.Join(root, "velox-host.exe")
	writeReleaseFile(t, cliPath, []byte("cli"))
	writeReleaseFile(t, hostPath, []byte("host"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "schema", "one.schema.json"), []byte("{}"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "THIRD_PARTY_NOTICES.md"), []byte("notices"))
	outputRoot := filepath.Join(root, "out")
	if _, err := Build(Options{CLIPath: cliPath, HostPath: hostPath, SourceRoot: sourceRoot, OutputRoot: outputRoot}); err != nil {
		t.Fatal(err)
	}
	first, err := Build(Options{CLIPath: cliPath, HostPath: hostPath, SourceRoot: sourceRoot, OutputRoot: outputRoot})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(Options{CLIPath: cliPath, HostPath: hostPath, SourceRoot: sourceRoot, OutputRoot: outputRoot})
	if err != nil {
		t.Fatal(err)
	}
	if first.ArchiveSHA256 != second.ArchiveSHA256 {
		t.Fatalf("replacement changed archive: %s != %s", first.ArchiveSHA256, second.ArchiveSHA256)
	}
}

func writeReleaseFile(t *testing.T, path string, value []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, value, 0o644); err != nil {
		t.Fatal(err)
	}
}
