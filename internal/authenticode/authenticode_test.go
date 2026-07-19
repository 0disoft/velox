package authenticode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSubject = "CN=Velox Test Publisher, O=0disoft, C=KR"

func TestVerifyDirectoryAcceptsOneSharedTrustedSigner(t *testing.T) {
	directory := signedFixture(t)
	result, err := verifyDirectory(directory, testSubject, func(_ context.Context, path string) (probeResult, error) {
		return validProbe(filepath.Base(path)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != SchemaVersion || result.Target != Target || result.ExpectedSubject != testSubject || len(result.Artifacts) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if result.Artifacts[0].File != "velox-host.exe" || result.Artifacts[1].File != "velox.exe" {
		t.Fatalf("artifact order = %#v", result.Artifacts)
	}
	for _, artifact := range result.Artifacts {
		if artifact.Status != "verified" || artifact.DigestAlgorithm != DigestName || artifact.DigestOID != DigestOID || artifact.TimestampAuthority == "" {
			t.Fatalf("artifact = %#v", artifact)
		}
	}
}

func TestVerifyDirectoryRejectsPolicyAndEvidenceFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*probeResult, string)
		want   string
	}{
		{"invalid status", func(result *probeResult, _ string) { result.Status = "HashMismatch" }, "status"},
		{"wrong subject", func(result *probeResult, _ string) { result.Subject = "CN=Other" }, "expected subject"},
		{"weak digest", func(result *probeResult, _ string) { result.DigestOID = "1.3.14.3.2.26" }, "not SHA-256"},
		{"missing timestamp", func(result *probeResult, _ string) { result.TimestampSubject = "" }, "timestamp authority"},
		{"different signer", func(result *probeResult, path string) {
			if path == "velox.exe" {
				result.Serial = "22"
			}
		}, "share one signer"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := verifyDirectory(signedFixture(t), testSubject, func(_ context.Context, path string) (probeResult, error) {
				result := validProbe(filepath.Base(path))
				test.mutate(&result, filepath.Base(path))
				return result, nil
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestVerifyDirectoryRejectsUnexpectedAndLinkedInputs(t *testing.T) {
	directory := signedFixture(t)
	writeFile(t, filepath.Join(directory, "provider.json"), "metadata")
	if _, err := verifyDirectory(directory, testSubject, nil); err == nil || !strings.Contains(err.Error(), "exactly") {
		t.Fatalf("extra file error = %v", err)
	}

	linked := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.exe")
	writeFile(t, target, "signed")
	if err := os.Symlink(target, filepath.Join(linked, "velox.exe")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	writeFile(t, filepath.Join(linked, "velox-host.exe"), "signed")
	if _, err := verifyDirectory(linked, testSubject, nil); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("linked file error = %v", err)
	}
}

func TestVerifyDirectoryPropagatesProbeFailure(t *testing.T) {
	_, err := verifyDirectory(signedFixture(t), testSubject, func(context.Context, string) (probeResult, error) {
		return probeResult{}, errors.New("probe unavailable")
	})
	if err == nil || !strings.Contains(err.Error(), "probe unavailable") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerificationSchemaIsDraft202012(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "authenticode-verification-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" || schema["$id"] != "https://schemas.velox.invalid/authenticode-verification/v1.json" {
		t.Fatalf("schema identity = %#v", schema)
	}
	encoded, err := json.Marshal(Result{
		SchemaVersion:   SchemaVersion,
		Target:          Target,
		ExpectedSubject: testSubject,
		Artifacts: []Artifact{
			mustArtifact(t, "velox-host.exe"),
			mustArtifact(t, "velox.exe"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{SchemaVersion, "velox-host.exe", "velox.exe", DigestOID, "timestampAuthority"} {
		if !strings.Contains(string(encoded), required) {
			t.Fatalf("encoded result lacks %q: %s", required, encoded)
		}
	}
}

func validProbe(string) probeResult {
	return probeResult{
		Status: "Valid", Subject: testSubject, Issuer: "CN=Velox Test CA",
		Serial: "10", Thumbprint: strings.Repeat("a", 40), DigestOID: DigestOID,
		TimestampSubject: "CN=Velox Test Timestamp", TimestampSerial: "20",
		TimestampThumbprint: strings.Repeat("b", 40),
	}
}

func mustArtifact(t *testing.T, file string) Artifact {
	t.Helper()
	artifact, err := validateProbe(file, testSubject, validProbe(file))
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func signedFixture(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "velox.exe"), "signed cli")
	writeFile(t, filepath.Join(directory, "velox-host.exe"), "signed host")
	return directory
}

func writeFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
