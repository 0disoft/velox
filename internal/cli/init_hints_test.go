package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/initializer"
)

func TestInitNextStepsQuoteLiteralPaths(t *testing.T) {
	for _, test := range []struct {
		name      string
		directory string
		goos      string
		argument  string
		shell     string
	}{
		{name: "default", directory: ".", goos: "windows", argument: "'velox.json'", shell: "PowerShell"},
		{name: "spaces", directory: "my editor", goos: "windows", argument: "'my editor/velox.json'", shell: "PowerShell"},
		{name: "powershell literals", directory: "parent's/$draft `note & editor", goos: "windows", argument: "'parent''s/$draft `note & editor/velox.json'", shell: "PowerShell"},
		{name: "posix spaces", directory: "my editor", goos: "linux", argument: "'my editor/velox.json'", shell: "POSIX shell"},
		{name: "posix literals", directory: "parent's/$draft `note & editor", goos: "linux", argument: "'parent'\"'\"'s/$draft `note & editor/velox.json'", shell: "POSIX shell"},
		{name: "macos", directory: "my editor", goos: "darwin", argument: "'my editor/velox.json'", shell: "POSIX shell"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			printInitNextSteps(&output, test.directory, test.goos)
			want := "\nNext steps (" + test.shell + "):\n  velox run --config " + test.argument + "\n  velox build --config " + test.argument + "\n"
			if output.String() != want {
				t.Fatalf("got %q, want %q", output.String(), want)
			}
		})
	}
}

func TestInitNextStepsPreserveModesAndProjectFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, mode := range []string{"human", "quiet", "json", "json-quiet"} {
		t.Run(mode, func(t *testing.T) {
			target := "my editor " + mode
			args := []string{"init", target}
			if strings.Contains(mode, "json") {
				args = append(args, "--json")
			}
			if strings.Contains(mode, "quiet") {
				args = append(args, "--quiet")
			}
			var stdout, stderr bytes.Buffer
			if code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr, GOOS: "windows"}); code != 0 || stderr.Len() != 0 {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			switch mode {
			case "human":
				for _, want := range []string{"Initialized dev.velox.my-editor-human in " + target, "Next steps (PowerShell):", "velox run --config '" + target + "/velox.json'", "velox build --config '" + target + "/velox.json'"} {
					if !strings.Contains(stdout.String(), want) {
						t.Fatalf("missing %q: %s", want, stdout.String())
					}
				}
			case "quiet":
				if stdout.Len() != 0 {
					t.Fatalf("quiet output: %q", stdout.String())
				}
			case "json", "json-quiet":
				var envelope struct {
					SchemaVersion int                `json:"schemaVersion"`
					OK            bool               `json:"ok"`
					Command       string             `json:"command"`
					Result        initializer.Result `json:"result"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				if !envelope.OK || envelope.SchemaVersion != 1 || envelope.Command != "init" || envelope.Result.Directory != target || strings.Contains(stdout.String(), "Next steps") {
					t.Fatalf("unexpected JSON: %s", stdout.String())
				}
			}
			baseline := filepath.Join(t.TempDir(), target)
			result, err := initializer.Create(baseline)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range result.Files {
				got, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(file)))
				if err != nil {
					t.Fatal(err)
				}
				want, err := os.ReadFile(filepath.Join(baseline, filepath.FromSlash(file)))
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("generated %s changed: %v", file, err)
				}
			}
			entries, err := os.ReadDir(target)
			if err != nil || len(entries) != 3 {
				t.Fatalf("unexpected project files: %v %v", entries, err)
			}
		})
	}
}

func TestInitFailureDoesNotPrintNextSteps(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, args := range [][]string{
		{"init", "sample", "--template", "unknown"},
		{"init", "sample", "extra"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, Dependencies{Stdout: &stdout, Stderr: &stderr}); code != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid init created files: %v %v", entries, err)
	}
}
