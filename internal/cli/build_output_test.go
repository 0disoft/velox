package cli

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildHumanOutputLocatesArtifactsOutsideWorkingDirectory(t *testing.T) {
	root, config, host := cliFixture(t)
	t.Chdir(t.TempDir())
	var stdout bytes.Buffer
	code := Run([]string{"build", "--config", config, "--out", "dist"}, Dependencies{
		Stdout: &stdout, Stderr: io.Discard, HostPath: host,
	})
	if code != 0 {
		t.Fatalf("build exit=%d output=%q", code, stdout.String())
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Directory: " + filepath.Join(root, "dist", "com.example.hello"),
		"Archive: " + filepath.Join(root, "dist", "com.example.hello.zip"),
	} {
		if !strings.Contains(stdout.String(), want+"\n") {
			t.Fatalf("missing usable output location %q in %q", want, stdout.String())
		}
	}
	stdout.Reset()
	if code := Run([]string{"build", "--config", config, "--out", "dist", "--quiet"}, Dependencies{
		Stdout: &stdout, Stderr: io.Discard, HostPath: host,
	}); code != 0 || stdout.Len() != 0 {
		t.Fatalf("quiet build exit=%d output=%q", code, stdout.String())
	}
}
