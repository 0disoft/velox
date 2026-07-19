package releaseevidence

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/velox/internal/releasebundle"
)

func TestBuildProducesDeterministicChecksumsSBOMAndProvenance(t *testing.T) {
	release := buildReleaseFixture(t)
	created := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	options := Options{ReleaseDirectory: release.Directory, ReleaseArchive: release.Archive, SourceRepository: "https://github.com/0disoft/velox", SourceCommit: strings.Repeat("a", 40), InvocationID: "test-1", CreatedAt: created}
	options.OutputRoot = filepath.Join(t.TempDir(), "first")
	first, err := Build(options)
	if err != nil {
		t.Fatal(err)
	}
	options.OutputRoot = filepath.Join(t.TempDir(), "second")
	second, err := Build(options)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{first.SBOM, second.SBOM}, {first.Provenance, second.Provenance}, {first.Checksums, second.Checksums}} {
		left, _ := os.ReadFile(pair[0])
		right, _ := os.ReadFile(pair[1])
		if !bytes.Equal(left, right) {
			t.Fatalf("release evidence differs: %s", filepath.Base(pair[0]))
		}
	}
	var document spdxDocument
	data, err := os.ReadFile(first.SBOM)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.SPDXVersion != "SPDX-2.3" || len(document.Packages) != 1 || len(document.Files) == 0 {
		t.Fatalf("unexpected SPDX document: %+v", document)
	}
	provenance, err := os.ReadFile(first.Provenance)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(provenance, []byte{'\n'}) != 1 {
		t.Fatal("provenance JSONL must contain exactly one compact statement")
	}
}

func TestBuildRejectsArtifactTampering(t *testing.T) {
	release := buildReleaseFixture(t)
	if err := os.WriteFile(filepath.Join(release.Directory, "velox.exe"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Build(Options{ReleaseDirectory: release.Directory, ReleaseArchive: release.Archive, OutputRoot: filepath.Join(t.TempDir(), "evidence"), SourceRepository: "https://github.com/0disoft/velox", SourceCommit: strings.Repeat("a", 40), InvocationID: "test-2", CreatedAt: time.Now().UTC()})
	if err == nil || !strings.Contains(err.Error(), "differs from manifest") {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
}

func buildReleaseFixture(t *testing.T) releasebundle.Result {
	t.Helper()
	root := t.TempDir()
	input := filepath.Join(root, "input")
	source := filepath.Join(root, "source")
	writeEvidenceFixture(t, filepath.Join(input, "velox.exe"), []byte("cli"))
	writeEvidenceFixture(t, filepath.Join(input, "velox-host.exe"), []byte("host"))
	for _, name := range []string{"build-result-v1.schema.json", "consumer-clean-v1.schema.json", "host-metadata-v1.schema.json", "ipc-v1.schema.json", "public-preview-verification-v1.schema.json", "release-manifest-v1.schema.json", "runtime-config-v1.schema.json", "velox-v1.schema.json"} {
		writeEvidenceFixture(t, filepath.Join(source, "schema", name), []byte("{}\n"))
	}
	writeEvidenceFixture(t, filepath.Join(source, "THIRD_PARTY_NOTICES.md"), []byte("notices\n"))
	result, err := releasebundle.Build(releasebundle.Options{CLIPath: filepath.Join(input, "velox.exe"), HostPath: filepath.Join(input, "velox-host.exe"), SourceRoot: source, OutputRoot: filepath.Join(root, "release")})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writeEvidenceFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
