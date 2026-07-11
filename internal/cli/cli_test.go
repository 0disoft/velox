package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateJSONContract(t *testing.T) {
	root, config, host := cliFixture(t)
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"validate", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
	})
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", exitCode, stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Command != "validate" || envelope.Diagnostics == nil {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestBuildJSONContractAndOutputs(t *testing.T) {
	root, config, host := cliFixture(t)
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"build", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
	})
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q stdout=%q", exitCode, stderr.String(), stdout.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	resultData, err := json.Marshal(envelope.Result)
	if err != nil {
		t.Fatal(err)
	}
	var result BuildResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		t.Fatal(err)
	}
	if result.ArchiveSHA256 == "" || !strings.HasSuffix(result.Archive, "hello.zip") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFailureJSONDoesNotExposeAbsolutePath(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "private", "velox.json")
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"validate", "--config", missing, "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: filepath.Join(root, "velox-host.exe"),
	})
	if exitCode != 2 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", exitCode, stderr.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte(root)) {
		t.Fatalf("JSON exposed absolute path: %s", stdout.Bytes())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "MANIFEST_INVALID" {
		t.Fatalf("unexpected failure: %+v", envelope)
	}
}

func TestValidateRejectsHostTampering(t *testing.T) {
	root, config, host := cliFixture(t)
	if err := os.WriteFile(host, []byte("tampered-host"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"validate", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
	})
	if exitCode != 4 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", exitCode, stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "HOST_INCOMPATIBLE" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
		t.Fatalf("validation failure created output: %v", err)
	}
}

func TestSubcommandHelpSucceeds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"validate", "--help"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if exitCode != 0 || !strings.Contains(stderr.String(), "-config") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func TestVersionJSONContract(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"version", "--json"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", exitCode, stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Command != "version" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestUsageFailureHonorsJSONAnywhere(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"validate", "--unknown", "--json"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if exitCode != 2 || !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func TestInspectJSONContract(t *testing.T) {
	root, config, host := cliFixture(t)
	var buildOut, buildErr bytes.Buffer
	if exitCode := Run([]string{"build", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{Stdout: &buildOut, Stderr: &buildErr, HostPath: host}); exitCode != 0 {
		t.Fatalf("build exit=%d stderr=%q", exitCode, buildErr.String())
	}
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"inspect", filepath.Join(root, "dist", "hello.zip"), "--json"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("inspect exit=%d stderr=%q", exitCode, stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Command != "inspect" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func cliFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	config := filepath.Join(root, "velox.json")
	host := filepath.Join(root, "release", "velox-host.exe")
	writeCLIFile(t, host, "host")
	digest := sha256.Sum256([]byte("host"))
	writeCLIFile(t, filepath.Join(filepath.Dir(host), "velox-host.json"), fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.1.0-dev","target":"windows-x64","contracts":{"host":1,"runtime":1},"host":{"file":"velox-host.exe","bytes":4,"sha256":"%x"}}`, digest))
	writeCLIFile(t, filepath.Join(root, "web", "index.html"), "<title>Hello</title>")
	writeCLIFile(t, config, `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1.0.0"}}`)
	return root, config, host
}

func writeCLIFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
