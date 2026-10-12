package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanFailureRecoveryGuidance(t *testing.T) {
	for _, scenario := range []string{"missing manifest", "syntax", "type", "permission value", "missing assets", "output overlap", "target"} {
		for _, jsonMode := range []bool{false, true} {
			t.Run(scenario+map[bool]string{false: "/human", true: "/json"}[jsonMode], func(t *testing.T) {
				root, config, host := cliFixture(t)
				args := []string{"validate", "--config", config}
				wantCode, wantHint := 2, "Check the manifest fields against the schema"
				switch scenario {
				case "missing manifest":
					if err := os.Remove(config); err != nil {
						t.Fatal(err)
					}
					wantHint = "Check --config"
				case "syntax":
					writeCLIFile(t, config, "{\n  \"schemaVersion\": 1,\n  \"app\": }\n")
					wantHint = "Correct the JSON syntax"
				case "type":
					writeCLIFile(t, config, "{\n  \"schemaVersion\": \"PRIVATE_INPUT_VALUE\"\n}")
					wantHint = "Check JSON value types"
				case "permission value":
					writeCLIFile(t, config, `{"schemaVersion":1,"app":{"id":"com.example.hello","name":"Hello","version":"1.0.0"},"security":{"permissions":["PRIVATE_INPUT_VALUE"]}}`)
				case "missing assets":
					if err := os.Remove(filepath.Join(root, "web", "index.html")); err != nil {
						t.Fatal(err)
					}
					wantCode, wantHint = 3, "Check assets.root and assets.entry"
				case "output overlap":
					args = append(args, "--out", "web")
					wantHint = "Set --out outside assets.root"
				case "target":
					args = append(args, "--target", "PRIVATE_INPUT_VALUE")
					wantHint = "Use --target windows-x64"
				}
				if jsonMode {
					args = append(args, "--json")
				}
				before, readErr := os.ReadFile(config)
				var stdout, stderr bytes.Buffer
				code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr, HostPath: host})
				if code != wantCode {
					t.Fatalf("exit %d: %s %s", code, &stdout, &stderr)
				}
				output := stderr.String()
				if jsonMode {
					if stderr.Len() != 0 {
						t.Fatalf("JSON stderr: %s", &stderr)
					}
					var envelope Envelope
					if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
						t.Fatal(err)
					}
					if envelope.OK || envelope.Error == nil || len(envelope.Diagnostics) != 1 {
						t.Fatalf("envelope: %+v", envelope)
					}
					output = envelope.Diagnostics[0].Message
					if scenario == "syntax" && envelope.Diagnostics[0].Line != 3 {
						t.Fatalf("location: %+v", envelope.Diagnostics[0])
					}
				} else {
					if stdout.Len() != 0 {
						t.Fatalf("human stdout: %s", &stdout)
					}
					if !strings.Contains(output, "velox.json") {
						t.Fatalf("missing location: %s", output)
					}
					if scenario == "syntax" && !strings.Contains(output, ":3:") {
						t.Fatalf("missing syntax location: %s", output)
					}
				}
				if !strings.Contains(output, wantHint) {
					t.Fatalf("missing %q: %s", wantHint, output)
				}
				if strings.Contains(stdout.String()+stderr.String(), root) || strings.Contains(stdout.String()+stderr.String(), "PRIVATE_INPUT_VALUE") {
					t.Fatalf("private input leaked: %s %s", &stdout, &stderr)
				}
				after, err := os.ReadFile(config)
				if readErr == nil && (err != nil || !bytes.Equal(before, after)) {
					t.Fatal("failure changed manifest")
				}
				if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
					t.Fatalf("failure created output: %v", err)
				}
			})
		}
	}
}

func TestUsageFailureDoesNotEchoFlagValues(t *testing.T) {
	for _, command := range []string{"init", "templates", "validate", "doctor", "run", "build", "inspect", "version"} {
		for _, jsonMode := range []bool{false, true} {
			t.Run(command+map[bool]string{false: "/human", true: "/json"}[jsonMode], func(t *testing.T) {
				args := []string{command, "--quiet=PRIVATE_INPUT_VALUE"}
				if jsonMode {
					args = append(args, "--json")
				}
				var stdout, stderr bytes.Buffer
				if code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != 2 {
					t.Fatalf("exit=%d", code)
				}
				output := stdout.String() + stderr.String()
				if strings.Contains(output, "PRIVATE_INPUT_VALUE") {
					t.Fatalf("flag value leaked: %s", output)
				}
				if !strings.Contains(output, "velox "+command+" --help") {
					t.Fatalf("no correction guidance: %s", output)
				}
				if jsonMode && (!json.Valid(stdout.Bytes()) || stderr.Len() != 0) {
					t.Fatalf("JSON channels: %s %s", &stdout, &stderr)
				}
			})
		}
	}
}

func TestHumanFailureDoesNotPrintArbitraryUnderlyingError(t *testing.T) {
	var stderr bytes.Buffer
	code := emitFailure(Dependencies{Stderr: &stderr}, "build", false, 6, "PACKAGING_FAILED", "Application packaging failed.", errors.New("PRIVATE_INPUT_VALUE"))
	if code != 6 || strings.Contains(stderr.String(), "PRIVATE_INPUT_VALUE") || !strings.Contains(stderr.String(), "Application packaging failed") {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
}

func TestUnknownCommandDoesNotEchoSuppliedText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"PRIVATE_INPUT_VALUE"}, Dependencies{Stdout: &stdout, Stderr: &stderr})
	if code != 2 || stdout.Len() != 0 || strings.Contains(stderr.String(), "PRIVATE_INPUT_VALUE") ||
		!strings.Contains(stderr.String(), "unknown command") || !strings.Contains(stderr.String(), "Commands:") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestManifestCorrectionCanRetryBuildAndInspect(t *testing.T) {
	root, config, host := cliFixture(t)
	valid, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	writeCLIFile(t, config, "{\n  \"schemaVersion\": }\n")
	var stdout, stderr bytes.Buffer
	dependencies := Dependencies{Stdout: &stdout, Stderr: &stderr, HostPath: host}
	args := []string{"build", "--config", config, "--json"}
	if code := Run(args, dependencies); code != 2 {
		t.Fatalf("invalid build exit=%d", code)
	}
	if !strings.Contains(stdout.String(), "Correct the JSON syntax") {
		t.Fatalf("no repair guidance: %s", &stdout)
	}
	if err := os.WriteFile(config, valid, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(args, dependencies); code != 0 {
		t.Fatalf("retry exit=%d: %s %s", code, &stdout, &stderr)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"inspect", filepath.Join(root, "dist", "com.example.hello.zip"), "--json"}, dependencies); code != 0 {
		t.Fatalf("inspect exit=%d: %s %s", code, &stdout, &stderr)
	}
}
