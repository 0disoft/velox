package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestGlobalHelpAndUsageChannels(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{name: "help", args: []string{"help"}},
		{name: "long help", args: []string{"--help"}},
		{name: "short help", args: []string{"-h"}},
		{name: "missing command", code: 2},
		{name: "unknown command", args: []string{"unknown"}, code: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(test.args, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != test.code {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			output := stdout.String()
			if test.code == 0 {
				if stderr.Len() != 0 {
					t.Fatalf("help wrote stderr: %q", stderr.String())
				}
			} else {
				if stdout.Len() != 0 {
					t.Fatalf("usage failure wrote stdout: %q", stdout.String())
				}
				output = stderr.String()
			}
			for _, want := range []string{
				"Usage: velox <init|templates|validate|doctor|run|build|inspect|version> [options]",
				"Commands:", "init [directory]", "templates", "validate", "doctor",
				"run", "build", "inspect <path>", "version", "Examples:",
				"velox templates", "velox init my-editor --template text-editor",
				"velox run --config my-editor/velox.json", "velox build --config my-editor/velox.json",
				"velox <command> --help",
			} {
				if !strings.Contains(output, want) {
					t.Fatalf("missing %q in help: %s", want, output)
				}
			}
		})
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("help created files: %v %v", entries, err)
	}
}

func TestInitHelpGuidesTemplateDiscoveryWithoutCreatingProject(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, args := range [][]string{
		{"init", "--help"},
		{"init", "-h"},
		{"init", "my-editor", "--help"},
		{"init", "--help", "--json"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != 0 || stdout.Len() != 0 {
			t.Fatalf("%v: exit=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
		if jsonRequested(args) {
			if stderr.Len() != 0 {
				t.Fatalf("JSON help wrote stderr: %q", stderr.String())
			}
		} else {
			for _, want := range []string{
				"Usage: velox init [directory] [options]", "template defaults to basic",
				"velox templates", "velox init my-editor --template text-editor",
				"Options:", "-template", "-json", "-quiet",
			} {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("missing %q in init help: %s", want, stderr.String())
				}
			}
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("init help created files: %v %v", entries, err)
	}
}
