package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/releasebundle"
	"github.com/0disoft/velox/internal/signingrecord"
)

func TestDryRunAndVerifyCommands(t *testing.T) {
	fixture := commandFixture(t)
	recordPath := filepath.Join(fixture.root, "output", "signing-record.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(append([]string{"dry-run"}, fixture.args(recordPath)...), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("dry-run code = %d, stderr = %s", code, stderr.String())
	}
	var created struct {
		SchemaVersion string `json:"schemaVersion"`
		Publishable   bool   `json:"publishable"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.SchemaVersion != "velox.signing-record-result/v1" || created.Publishable {
		t.Fatalf("dry-run output = %#v", created)
	}
	record, err := signingrecord.DecodeFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if record.Mode != signingrecord.ModeDryRun || record.Publishable {
		t.Fatalf("record = %#v", record)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(append([]string{"verify", "--record", recordPath}, fixture.pathArgs()...), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("verify code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid":true`) || !strings.Contains(stdout.String(), `"publishable":false`) {
		t.Fatalf("verify output = %s", stdout.String())
	}
}

func TestPrepareCommandCreatesSigningInput(t *testing.T) {
	root := t.TempDir()
	unsigned := filepath.Join(root, "unsigned")
	writeCommandFile(t, filepath.Join(unsigned, "velox.exe"), "unsigned cli")
	writeCommandFile(t, filepath.Join(unsigned, "velox-host.exe"), "unsigned host")
	out := filepath.Join(root, "input", signingrecord.SigningInputName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"prepare", "--unsigned-dir", unsigned, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("prepare code = %d, stderr = %s", code, stderr.String())
	}
	var created struct {
		SchemaVersion string                           `json:"schemaVersion"`
		Command       string                           `json:"command"`
		Publishable   bool                             `json:"publishable"`
		Result        signingrecord.SigningInputResult `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.SchemaVersion != "velox.signing-record-result/v1" || created.Command != "prepare" || created.Publishable || created.Result.Path != out || created.Result.Artifact.File != signingrecord.SigningInputName {
		t.Fatalf("prepare output = %#v", created)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"prepare", "--unsigned-dir", unsigned, "--out", out}, &stdout, &stderr)
	if code != 6 || !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("second prepare code = %d, stderr = %s", code, stderr.String())
	}
}

func TestVerifyCommandRejectsChangedEvidence(t *testing.T) {
	fixture := commandFixture(t)
	recordPath := filepath.Join(fixture.root, "signing-record.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(append([]string{"dry-run"}, fixture.args(recordPath)...), &stdout, &stderr); code != 0 {
		t.Fatalf("dry-run code = %d, stderr = %s", code, stderr.String())
	}
	if err := os.WriteFile(filepath.Join(fixture.signed, "velox.exe"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code := run(append([]string{"verify", "--record", recordPath}, fixture.pathArgs()...), &stdout, &stderr)
	if code != 6 || !strings.Contains(stderr.String(), "signed CLI differs") {
		t.Fatalf("verify code = %d, stderr = %s", code, stderr.String())
	}
}

func TestCommandRejectsIncompleteArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"dry-run", "--out", "record.json"}, &stdout, &stderr); code != 2 {
		t.Fatalf("dry-run code = %d", code)
	}
	stderr.Reset()
	if code := run([]string{"verify"}, &stdout, &stderr); code != 2 {
		t.Fatalf("verify code = %d", code)
	}
	stderr.Reset()
	if code := run([]string{"prepare", "--out", signingrecord.SigningInputName}, &stdout, &stderr); code != 2 {
		t.Fatalf("prepare code = %d", code)
	}
}

type signingCommandFixture struct {
	root           string
	unsigned       string
	signingInput   string
	signed         string
	release        string
	releaseArchive string
	evidence       string
}

func commandFixture(t *testing.T) signingCommandFixture {
	t.Helper()
	root := t.TempDir()
	fixture := signingCommandFixture{
		root:           root,
		unsigned:       filepath.Join(root, "unsigned"),
		signingInput:   filepath.Join(root, "velox-signing-input.zip"),
		signed:         filepath.Join(root, "signed"),
		release:        filepath.Join(root, "release", "velox-windows-x64"),
		releaseArchive: filepath.Join(root, "release", "velox-windows-x64.zip"),
		evidence:       filepath.Join(root, "evidence"),
	}
	files := map[string]string{
		filepath.Join(fixture.unsigned, "velox.exe"):      "unsigned cli",
		filepath.Join(fixture.unsigned, "velox-host.exe"): "unsigned host",
		filepath.Join(fixture.signed, "velox.exe"):        "signed cli",
		filepath.Join(fixture.signed, "velox-host.exe"):   "signed host",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	unsignedCLI := filepath.Join(fixture.unsigned, "velox.exe")
	unsignedHost := filepath.Join(fixture.unsigned, "velox-host.exe")
	writeCommandZIP(t, fixture.signingInput, []commandZIPFile{{Name: "velox-host.exe", Path: unsignedHost}, {Name: "velox.exe", Path: unsignedCLI}})
	signedCLIPath := filepath.Join(fixture.signed, "velox.exe")
	signedHostPath := filepath.Join(fixture.signed, "velox-host.exe")
	signedCLI := commandArtifact(t, signedCLIPath, "velox.exe")
	signedHost := commandArtifact(t, signedHostPath, "velox-host.exe")
	manifestPath := filepath.Join(fixture.release, "release-manifest.json")
	writeCommandJSON(t, manifestPath, releasebundle.Manifest{
		SchemaVersion:  releasebundle.SchemaVersion,
		ReleaseVersion: "0.5.7-dev",
		Target:         signingrecord.Target,
		Artifacts: []releasebundle.Artifact{
			{File: signedCLI.File, Bytes: signedCLI.Bytes, SHA256: signedCLI.SHA256},
			{File: signedHost.File, Bytes: signedHost.Bytes, SHA256: signedHost.SHA256},
		},
	})
	writeCommandZIP(t, fixture.releaseArchive, []commandZIPFile{
		{Name: "velox-windows-x64/velox.exe", Path: signedCLIPath},
		{Name: "velox-windows-x64/velox-host.exe", Path: signedHostPath},
		{Name: "velox-windows-x64/release-manifest.json", Path: manifestPath},
	})
	archive := commandArtifact(t, fixture.releaseArchive, "velox-windows-x64.zip")
	sbomPath := filepath.Join(fixture.evidence, "velox-windows-x64.spdx.json")
	writeCommandJSON(t, sbomPath, map[string]any{"spdxVersion": "SPDX-2.3", "documentNamespace": "https://github.com/0disoft/velox/sbom/test/" + archive.SHA256})
	sbom := commandArtifact(t, sbomPath, "velox-windows-x64.spdx.json")
	writeCommandFile(t, filepath.Join(fixture.evidence, "checksums.sha256"), fmt.Sprintf("%s  %s\n%s  %s\n", archive.SHA256, archive.File, sbom.SHA256, sbom.File))
	return fixture
}

func (fixture signingCommandFixture) pathArgs() []string {
	return []string{
		"--unsigned-dir", fixture.unsigned,
		"--signing-input", fixture.signingInput,
		"--signed-dir", fixture.signed,
		"--release-dir", fixture.release,
		"--release-archive", fixture.releaseArchive,
		"--evidence-dir", fixture.evidence,
	}
}

func (fixture signingCommandFixture) args(recordPath string) []string {
	return append([]string{
		"--out", recordPath,
		"--release-version", "0.5.7-dev",
		"--source-commit", strings.Repeat("a", 40),
		"--source-tag", "v0.5.6-alpha.1",
		"--source-workflow", ".github/workflows/release.yml@refs/tags/v0.5.6-alpha.1",
		"--source-run-id", "12345",
		"--provider-project", "velox-dry-run",
		"--artifact-configuration", "windows-x64-dry-run",
		"--signing-policy", "test-signing-policy",
		"--request-id", "dry-run-12345",
	}, fixture.pathArgs()...)
}

type commandZIPFile struct {
	Name string
	Path string
}

func writeCommandZIP(t *testing.T, path string, files []commandZIPFile) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(output)
	for _, file := range files {
		data, err := os.ReadFile(file.Path)
		if err != nil {
			t.Fatal(err)
		}
		entry, err := archive.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeCommandJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeCommandFile(t, path, string(data)+"\n")
}

func writeCommandFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commandArtifact(t *testing.T, path, name string) signingrecord.Artifact {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return signingrecord.Artifact{File: name, Bytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}
}
