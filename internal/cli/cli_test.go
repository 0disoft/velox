package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func TestSuccessfulJSONWriteFailureReturnsInternalExit(t *testing.T) {
	root, config, host := cliFixture(t)
	exitCode := Run([]string{"validate", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: failingWriter{}, Stderr: io.Discard, HostPath: host,
	})
	if exitCode != 10 {
		t.Fatalf("exit = %d, want 10", exitCode)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

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
	if result.ArchiveSHA256 == "" || !strings.HasSuffix(result.Archive, "com.example.hello.zip") {
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
	encoded, err := json.Marshal(envelope.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"ipcVersions":[1]`) {
		t.Fatalf("version result does not advertise IPC v1: %s", encoded)
	}
}

func TestUsageFailureHonorsJSONAnywhere(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"validate", "--unknown", "--json"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if exitCode != 2 || !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func TestInitJSONContract(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample-app")
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"init", target, "--json"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Command != "init" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if _, err := os.Stat(filepath.Join(target, "velox.json")); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorJSONContract(t *testing.T) {
	root, config, host := cliFixture(t)
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"doctor", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
		GOOS: "windows", GOARCH: "amd64", WebView2VersionProbe: func() (string, error) { return "123.0.0.0", nil },
	})
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Command != "doctor" || envelope.Diagnostics == nil {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestDoctorFailureIncludesChecksWithoutExposingProbeError(t *testing.T) {
	root, config, host := cliFixture(t)
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"doctor", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
		GOOS: "windows", GOARCH: "amd64", WebView2VersionProbe: func() (string, error) { return "", errors.New(`private C:\runtime\probe failed`) },
	})
	if exitCode != 5 || stderr.Len() != 0 || bytes.Contains(stdout.Bytes(), []byte(`C:\runtime`)) {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "RUNTIME_WEBVIEW2_PROBE_FAILED" || envelope.Result == nil {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestRunJSONContractAndTemporaryConfigCleanup(t *testing.T) {
	root, config, host := cliFixture(t)
	var configPath string
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"run", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
		HostLauncher: func(hostPath, runtimeConfig string, childStdout, childStderr io.Writer) (int, error) {
			configPath = runtimeConfig
			if childStdout != io.Discard || childStderr != io.Discard {
				t.Fatal("JSON mode exposed child output streams")
			}
			if _, err := os.Stat(runtimeConfig); err != nil {
				t.Fatal(err)
			}
			return 0, nil
		},
	})
	if exitCode != 0 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("temporary config remained: %v", err)
	}
}

func TestRunPreservesHostExitCode(t *testing.T) {
	root, config, host := cliFixture(t)
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"run", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
		HostLauncher: func(hostPath, runtimeConfig string, childStdout, childStderr io.Writer) (int, error) {
			return 5, nil
		},
	})
	if exitCode != 5 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var envelope Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "RUNTIME_HOST_EXITED" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestInspectJSONContract(t *testing.T) {
	root, config, host := cliFixture(t)
	var buildOut, buildErr bytes.Buffer
	if exitCode := Run([]string{"build", "--config", config, "--out", filepath.Join(root, "dist"), "--json"}, Dependencies{Stdout: &buildOut, Stderr: &buildErr, HostPath: host}); exitCode != 0 {
		t.Fatalf("build exit=%d stderr=%q", exitCode, buildErr.String())
	}
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"inspect", filepath.Join(root, "dist", "com.example.hello.zip"), "--json"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
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
	writeCLIFile(t, filepath.Join(filepath.Dir(host), "velox-host.json"), fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":"0.5.2-dev","target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"velox-host.exe","bytes":4,"sha256":"%x"}}`, digest))
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
