package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/initializer"
)

func TestTemplatesOutputAndReadOnly(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, mode := range []string{"human", "quiet", "json", "json-quiet", "help"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"templates"}
			switch mode {
			case "quiet":
				args = append(args, "--quiet")
			case "json":
				args = append(args, "--json")
			case "json-quiet":
				args = append(args, "--json", "--quiet")
			case "help":
				args = append(args, "--help")
			}
			var stdout, stderr bytes.Buffer
			if code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != 0 {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if mode == "help" {
				if stdout.Len() != 0 || !strings.Contains(stderr.String(), "-json") || !strings.Contains(stderr.String(), "-quiet") {
					t.Fatalf("unexpected help: stdout=%q stderr=%q", stdout.String(), stderr.String())
				}
			} else {
				if stderr.Len() != 0 {
					t.Fatalf("unexpected stderr: %s", stderr.String())
				}
				switch mode {
				case "quiet":
					if stdout.Len() != 0 {
						t.Fatalf("quiet output: %s", stdout.String())
					}
				case "json", "json-quiet":
					var envelope struct {
						SchemaVersion int             `json:"schemaVersion"`
						OK            bool            `json:"ok"`
						Command       string          `json:"command"`
						Result        TemplatesResult `json:"result"`
						Diagnostics   []Diagnostic    `json:"diagnostics"`
					}
					if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
						t.Fatal(err)
					}
					if !envelope.OK || envelope.SchemaVersion != 1 || envelope.Command != "templates" || envelope.Diagnostics == nil || len(envelope.Diagnostics) != 0 || !reflect.DeepEqual(envelope.Result.Templates, initializer.Templates()) {
						t.Fatalf("unexpected envelope: %+v", envelope)
					}
				case "human":
					for _, template := range initializer.Templates() {
						for _, want := range append([]string{template.Name, template.Description, template.InitCommand}, template.Permissions...) {
							if !strings.Contains(stdout.String(), want) {
								t.Fatalf("missing %q: %s", want, stdout.String())
							}
						}
					}
					if !strings.Contains(stdout.String(), "Permissions: none") {
						t.Fatal("basic template lacks explicit empty permissions")
					}
				}
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("templates created files: %v %v", entries, err)
			}
		})
	}
}

func TestTemplatesRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"templates", "--unknown"},
		{"templates", "extra"},
		{"templates", "--unknown", "--json"},
		{"templates", "extra", "--json"},
		{"templates", "--json", "extra"},
		{"templates", "--config", "velox.json", "--json"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != 2 {
			t.Fatalf("%v: exit=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
		if jsonRequested(args) {
			var envelope Envelope
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.OK || envelope.Command != "templates" || envelope.Error == nil || envelope.Error.Code != "USAGE_INVALID" || stderr.Len() != 0 {
				t.Fatalf("unexpected failure: %+v stderr=%q", envelope, stderr.String())
			}
		} else if stdout.Len() != 0 || !strings.Contains(stderr.String(), "USAGE_INVALID") {
			t.Fatalf("unexpected failure: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	}
}

func TestUsageIncludesTemplates(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"help"}, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != 0 || !strings.Contains(stdout.String(), "|templates|") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
