package signingrecord

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
)

func TestBuildDryRunWritesDeterministicNonPublishableRecord(t *testing.T) {
	options := dryRunFixture(t)
	first, err := BuildDryRun(options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildDryRun(options)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("dry-run signing record is not deterministic")
	}
	if first.Publishable || first.Certificate.Status != StatusNotPerformed || first.Certificate.Subject != "" {
		t.Fatal("dry-run signing record claimed publishable certificate evidence")
	}
	for _, attestation := range first.Attestations {
		if attestation.Status != StatusNotPerformed {
			t.Fatalf("dry-run attestation status = %q", attestation.Status)
		}
	}

	path := filepath.Join(t.TempDir(), "signing-record.json")
	result, err := Write(path, first)
	if err != nil {
		t.Fatal(err)
	}
	if result.SHA256 == "" || result.Path != path {
		t.Fatalf("write result = %#v", result)
	}
	decoded, err := DecodeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyFiles(decoded, options.Files); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFilesRejectsTampering(t *testing.T) {
	options := dryRunFixture(t)
	record, err := BuildDryRun(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.Files.SignedHost, []byte("tampered signed host"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = VerifyFiles(record, options.Files)
	if err == nil || !strings.Contains(err.Error(), "signed host differs") {
		t.Fatalf("VerifyFiles error = %v", err)
	}
}

func TestBuildDryRunRejectsUnexpectedProviderOutput(t *testing.T) {
	fixture := dryRunFixture(t)
	extra := filepath.Join(filepath.Dir(fixture.Files.SignedCLI), "provider-response.json")
	writeTestFile(t, extra, "untrusted provider metadata")
	if _, err := BuildDryRun(fixture); err == nil || !strings.Contains(err.Error(), "must contain exactly two entries") {
		t.Fatalf("BuildDryRun error = %v", err)
	}
}

func TestBuildDryRunRejectsSplitProviderOutputDirectories(t *testing.T) {
	fixture := dryRunFixture(t)
	otherDirectory := filepath.Join(filepath.Dir(filepath.Dir(fixture.Files.SignedHost)), "other-signed")
	otherHost := filepath.Join(otherDirectory, "velox-host.exe")
	writeTestFile(t, otherHost, "signed host")
	fixture.Files.SignedHost = otherHost
	if _, err := BuildDryRun(fixture); err == nil || !strings.Contains(err.Error(), "must share one directory") {
		t.Fatalf("BuildDryRun error = %v", err)
	}
}

func TestBuildDryRunRejectsUnchangedSignedArtifact(t *testing.T) {
	options := dryRunFixture(t)
	unsigned, err := os.ReadFile(options.Files.UnsignedCLI)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.Files.SignedCLI, unsigned, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = BuildDryRun(options)
	if err == nil || !strings.Contains(err.Error(), "has the unsigned digest") {
		t.Fatalf("BuildDryRun error = %v", err)
	}
}

func TestBuildDryRunRejectsBrokenArtifactLineage(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*testing.T, DryRunOptions)
		message string
	}{
		{
			name: "signing input",
			mutate: func(t *testing.T, options DryRunOptions) {
				writeTestZIP(t, options.Files.SigningInput, []testZIPFile{
					{Name: "velox.exe", Path: options.Files.SignedCLI},
					{Name: "velox-host.exe", Path: options.Files.UnsignedHost},
				})
			},
			message: "verify signing input",
		},
		{
			name: "release manifest",
			mutate: func(t *testing.T, options DryRunOptions) {
				data, err := os.ReadFile(options.Files.ReleaseManifest)
				if err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, options.Files.ReleaseManifest, strings.Replace(string(data), "0.5.9-dev", "0.5.10-dev", 1))
			},
			message: "verify final release manifest",
		},
		{
			name: "release archive",
			mutate: func(t *testing.T, options DryRunOptions) {
				writeTestFile(t, options.Files.ReleaseArchive, "not a zip")
			},
			message: "verify final release archive",
		},
		{
			name: "checksums",
			mutate: func(t *testing.T, options DryRunOptions) {
				writeTestFile(t, options.Files.Checksums, strings.Repeat("0", 64)+"  velox-windows-x64.zip\n"+strings.Repeat("0", 64)+"  velox-windows-x64.spdx.json\n")
			},
			message: "verify final checksums",
		},
		{
			name: "SBOM",
			mutate: func(t *testing.T, options DryRunOptions) {
				writeTestJSON(t, options.Files.SBOM, map[string]any{"spdxVersion": "SPDX-2.3", "documentNamespace": "https://example.invalid/wrong"})
				archive := testArtifact(t, options.Files.ReleaseArchive, "velox-windows-x64.zip")
				sbom := testArtifact(t, options.Files.SBOM, "velox-windows-x64.spdx.json")
				writeTestFile(t, options.Files.Checksums, fmt.Sprintf("%s  %s\n%s  %s\n", archive.SHA256, archive.File, sbom.SHA256, sbom.File))
			},
			message: "verify final SBOM",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := dryRunFixture(t)
			test.mutate(t, options)
			_, err := BuildDryRun(options)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("BuildDryRun error = %v", err)
			}
		})
	}
}

func TestValidateReleaseRequiresVerifiedCertificateAndAttestations(t *testing.T) {
	record, err := BuildDryRun(dryRunFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	record.Mode = ModeRelease
	record.Publishable = true
	record.Certificate = Certificate{Status: StatusVerified, Subject: "CN=Velox", Issuer: "CN=Issuer", Serial: "01", TimestampAuthority: "CN=Timestamp"}
	for index := range record.Attestations {
		record.Attestations[index].Status = StatusVerified
	}
	if err := Validate(record); err != nil {
		t.Fatal(err)
	}
	record.Attestations[0].Status = StatusNotPerformed
	if err := Validate(record); err == nil || !strings.Contains(err.Error(), "must be verified") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestDecodeFileRejectsUnknownAndTrailingData(t *testing.T) {
	record, err := BuildDryRun(dryRunFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	unknown := bytes.Replace(data, []byte(`"schemaVersion":`), []byte(`"token":"must-not-be-accepted","schemaVersion":`), 1)
	path := filepath.Join(t.TempDir(), "unknown.json")
	if err := os.WriteFile(path, unknown, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeFile(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	path = filepath.Join(t.TempDir(), "trailing.json")
	if err := os.WriteFile(path, append(data, []byte("\n{}\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeFile(path); err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("trailing-data error = %v", err)
	}
}

func TestValidateRejectsCredentialBearingRepositoryURL(t *testing.T) {
	record, err := BuildDryRun(dryRunFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	record.Source.Repository = "https://token@example.invalid/velox"
	if err := Validate(record); err == nil || !strings.Contains(err.Error(), "without credentials") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestWriteRefusesExistingOutput(t *testing.T) {
	record, err := BuildDryRun(dryRunFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signing-record.json")
	if err := os.WriteFile(path, []byte("preserve"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(path, record); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Write error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "preserve" {
		t.Fatal("existing signing record output changed")
	}
}

func TestSigningRecordSchemaIsDraft202012(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "signing-record-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" || schema["$id"] != "https://schemas.velox.invalid/signing-record/v1.json" {
		t.Fatalf("unexpected signing record schema identity: %#v", schema)
	}
}

func TestDryRunSchemaFixture(t *testing.T) {
	output := os.Getenv("VELOX_SIGNING_RECORD_RESULT")
	if output == "" {
		t.Skip("VELOX_SIGNING_RECORD_RESULT is not set")
	}
	record, err := BuildDryRun(dryRunFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(output, record); err != nil {
		t.Fatal(err)
	}
}

func dryRunFixture(t *testing.T) DryRunOptions {
	t.Helper()
	root := t.TempDir()
	files := Files{
		UnsignedCLI:     filepath.Join(root, "unsigned", "velox.exe"),
		UnsignedHost:    filepath.Join(root, "unsigned", "velox-host.exe"),
		SigningInput:    filepath.Join(root, "velox-signing-input.zip"),
		SignedCLI:       filepath.Join(root, "signed", "velox.exe"),
		SignedHost:      filepath.Join(root, "signed", "velox-host.exe"),
		ReleaseArchive:  filepath.Join(root, "release", "velox-windows-x64.zip"),
		ReleaseManifest: filepath.Join(root, "release", "velox-windows-x64", "release-manifest.json"),
		Checksums:       filepath.Join(root, "evidence", "checksums.sha256"),
		SBOM:            filepath.Join(root, "evidence", "velox-windows-x64.spdx.json"),
	}
	contents := map[string]string{
		files.UnsignedCLI:  "unsigned cli",
		files.UnsignedHost: "unsigned host",
		files.SignedCLI:    "signed cli",
		files.SignedHost:   "signed host",
	}
	for path, content := range contents {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeTestZIP(t, files.SigningInput, []testZIPFile{
		{Name: "velox-host.exe", Path: files.UnsignedHost},
		{Name: "velox.exe", Path: files.UnsignedCLI},
	})
	signedCLI := testArtifact(t, files.SignedCLI, "velox.exe")
	signedHost := testArtifact(t, files.SignedHost, "velox-host.exe")
	manifest := releasebundle.Manifest{
		SchemaVersion:  releasebundle.SchemaVersion,
		ReleaseVersion: "0.5.9-dev",
		Target:         Target,
		Artifacts: []releasebundle.Artifact{
			{File: signedCLI.File, Bytes: signedCLI.Bytes, SHA256: signedCLI.SHA256},
			{File: signedHost.File, Bytes: signedHost.Bytes, SHA256: signedHost.SHA256},
		},
	}
	writeTestJSON(t, files.ReleaseManifest, manifest)
	writeTestZIP(t, files.ReleaseArchive, []testZIPFile{
		{Name: "velox-windows-x64/velox.exe", Path: files.SignedCLI},
		{Name: "velox-windows-x64/velox-host.exe", Path: files.SignedHost},
		{Name: "velox-windows-x64/release-manifest.json", Path: files.ReleaseManifest},
	})
	archive := testArtifact(t, files.ReleaseArchive, "velox-windows-x64.zip")
	writeTestJSON(t, files.SBOM, map[string]any{
		"spdxVersion":       "SPDX-2.3",
		"documentNamespace": "https://github.com/0disoft/velox/sbom/test/" + archive.SHA256,
	})
	sbom := testArtifact(t, files.SBOM, "velox-windows-x64.spdx.json")
	writeTestFile(t, files.Checksums, fmt.Sprintf("%s  %s\n%s  %s\n", archive.SHA256, archive.File, sbom.SHA256, sbom.File))
	return DryRunOptions{
		ReleaseVersion: "0.5.9-dev",
		Source: Source{
			Repository: "https://github.com/0disoft/velox",
			Commit:     strings.Repeat("a", 40),
			Tag:        "v0.5.6-alpha.1",
			Workflow:   ".github/workflows/release.yml@refs/tags/v0.5.6-alpha.1",
			RunID:      "12345",
		},
		Provider: Provider{
			Name:                  ProviderSignPath,
			Project:               "velox-dry-run",
			ArtifactConfiguration: "windows-x64-dry-run",
			SigningPolicy:         "test-signing-policy",
			RequestID:             "dry-run-12345",
		},
		Files: files,
	}
}

type testZIPFile struct {
	Name string
	Path string
}

func writeTestZIP(t *testing.T, path string, files []testZIPFile) {
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

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(data)+"\n")
}

func writeTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testArtifact(t *testing.T, path, name string) Artifact {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return Artifact{File: name, Bytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}
}
