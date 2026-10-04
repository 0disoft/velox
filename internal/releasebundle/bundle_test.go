package releasebundle

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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
	writeReleaseInputs(t, sourceRoot)
	writeReleaseFile(t, filepath.Join(sourceRoot, "schema", "consumer-e2e-v1.schema.json"), []byte("must-not-ship\n"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "schema", "signing-record-v1.schema.json"), []byte("must-not-ship\n"))
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
	if manifest.SchemaVersion != SchemaVersion || len(manifest.Artifacts) != 15 {
		t.Fatalf("unexpected release manifest: %+v", manifest)
	}
	if _, err := os.Stat(filepath.Join(first.Directory, "schema", "public-preview-verification-v1.schema.json")); err != nil {
		t.Fatalf("public-preview verification schema is absent from consumer release: %v", err)
	}
	for _, name := range []string{"consumer-e2e-v1.schema.json", "signing-record-v1.schema.json"} {
		if _, err := os.Stat(filepath.Join(first.Directory, "schema", name)); !os.IsNotExist(err) {
			t.Fatalf("maintainer-only schema %s shipped in consumer release: %v", name, err)
		}
	}
	for index := 1; index < len(manifest.Artifacts); index++ {
		if manifest.Artifacts[index-1].File >= manifest.Artifacts[index].File {
			t.Fatalf("release artifacts are not sorted: %+v", manifest.Artifacts)
		}
	}
	reader, err := zip.OpenReader(first.Archive)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, name := range releaseTypeFiles {
		relative := "types/" + name
		want, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, artifact := range manifest.Artifacts {
			if artifact.File == relative {
				found = artifact.Bytes == int64(len(want)) && artifact.SHA256 == fmt.Sprintf("%x", sha256.Sum256(want))
			}
		}
		if !found {
			t.Fatalf("missing or incorrect type artifact inventory: %s", relative)
		}
		found = false
		for _, entry := range reader.File {
			if entry.Name != "velox-windows-x64/"+relative {
				continue
			}
			file, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(file)
			file.Close()
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("type ZIP contents differ: %s %v", relative, err)
			}
			found = true
		}
		if !found {
			t.Fatalf("missing type ZIP entry: %s", relative)
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
	writeReleaseInputs(t, sourceRoot)
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

func TestOptionalSetupIsIncludedInReleaseInventory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	cli, host, setup := filepath.Join(root, "velox.exe"), filepath.Join(root, "host.exe"), filepath.Join(root, "setup.exe")
	writeReleaseFile(t, cli, []byte("cli"))
	writeReleaseFile(t, host, []byte("host"))
	writeReleaseFile(t, setup, []byte("setup"))
	writeReleaseInputs(t, source)
	writeReleaseFile(t, filepath.Join(source, "THIRD_PARTY_NOTICES.md"), []byte("notices"))
	result, err := Build(Options{CLIPath: cli, HostPath: host, SetupPath: setup, SourceRoot: source, OutputRoot: filepath.Join(root, "out")})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(result.Directory, "release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, file := range manifest.Artifacts {
		if file.File == "velox-setup.exe" && file.Bytes == 5 && file.SHA256 != "" {
			return
		}
	}
	t.Fatal("setup template missing from release inventory")
}

func TestBuildFailsWhenRequiredReleaseSchemaIsMissing(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	cliPath := filepath.Join(root, "velox.exe")
	hostPath := filepath.Join(root, "velox-host.exe")
	writeReleaseFile(t, cliPath, []byte("cli"))
	writeReleaseFile(t, hostPath, []byte("host"))
	writeReleaseFile(t, filepath.Join(sourceRoot, "THIRD_PARTY_NOTICES.md"), []byte("notices"))

	if _, err := Build(Options{CLIPath: cliPath, HostPath: hostPath, SourceRoot: sourceRoot, OutputRoot: filepath.Join(root, "out")}); err == nil {
		t.Fatal("expected missing required release schema to fail")
	}
}

func TestBuildFailsWhenRequiredTypeFileIsMissing(t *testing.T) {
	for _, name := range releaseTypeFiles {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "source")
			cli, host := filepath.Join(root, "cli.exe"), filepath.Join(root, "host.exe")
			writeReleaseFile(t, cli, []byte("cli"))
			writeReleaseFile(t, host, []byte("host"))
			writeReleaseInputs(t, source)
			writeReleaseFile(t, filepath.Join(source, "THIRD_PARTY_NOTICES.md"), []byte("notices"))
			if err := os.Remove(filepath.Join(source, "types", name)); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(root, "out")
			if _, err := Build(Options{CLIPath: cli, HostPath: host, SourceRoot: source, OutputRoot: out}); err == nil {
				t.Fatal("accepted missing type file")
			}
			entries, err := os.ReadDir(out)
			if err != nil || len(entries) != 0 {
				t.Fatalf("failed bundle left outputs: %v %v", entries, err)
			}
		})
	}
}

func writeReleaseInputs(t *testing.T, sourceRoot string) {
	t.Helper()
	for _, name := range releaseSchemaFiles {
		writeReleaseFile(t, filepath.Join(sourceRoot, "schema", name), []byte("{}\n"))
	}
	for _, name := range releaseTypeFiles {
		writeReleaseFile(t, filepath.Join(sourceRoot, "types", name), []byte("type fixture: "+name+"\n"))
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
