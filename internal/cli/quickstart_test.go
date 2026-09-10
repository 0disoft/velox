package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/doctor"
)

func TestQuickstartCLISequence(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "QUICKSTART.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(doc), "## 4. Exercise the public CLI")
	if !found {
		t.Fatal("missing quickstart CLI section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	root, _, host := cliFixture(t)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	var launchedConfig string
	deps := Dependencies{
		Stdout: &stdout, Stderr: &stderr, HostPath: host,
		GOOS: "windows", GOARCH: "amd64",
		WindowsVersionProbe: func() doctor.WindowsVersion {
			return doctor.WindowsVersion{Major: 10, Build: doctor.MinimumWindowsClientBuild}
		},
		WebView2VersionProbe: func() (string, error) { return "123.0.0.0", nil },
		HostLauncher: func(_ string, config string, _, _ io.Writer) (int, error) {
			launchedConfig = config
			_, err := os.Stat(config)
			return 0, err
		},
	}
	// These examples contain literal CLI arguments only. Never evaluate shell code.
	literal := regexp.MustCompile(`^[A-Za-z0-9_./\\-]+$`)
	var commands []string
	for _, line := range strings.Split(section, "\n") {
		argsText, ok := strings.CutPrefix(strings.TrimSpace(line), "& $Velox ")
		if !ok {
			continue
		}
		args := strings.Fields(argsText)
		for i, arg := range args {
			if !literal.MatchString(arg) {
				t.Fatalf("quickstart requires an explicit test update for argument %q", arg)
			}
			args[i] = strings.ReplaceAll(arg, `\`, "/")
		}
		stdout.Reset()
		stderr.Reset()
		if code := Run(args, deps); code != 0 {
			t.Fatalf("%s: exit=%d stdout=%s stderr=%s", line, code, &stdout, &stderr)
		}
		var result Envelope
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.OK {
			t.Fatalf("%s: invalid success response %s (%v)", line, &stdout, err)
		}
		commands = append(commands, args[0])
	}
	if got := strings.Join(commands, ","); got != "version,init,validate,doctor,build,inspect,run" {
		t.Fatalf("unexpected quickstart sequence: %s", got)
	}
	for _, output := range []string{"dev.velox.hello", "dev.velox.hello.zip"} {
		if _, err := os.Stat(filepath.Join(root, "work", "dist", output)); err != nil {
			t.Fatal(err)
		}
	}
	if launchedConfig == "" {
		t.Fatal("run did not invoke the launcher")
	}
	if _, err := os.Stat(launchedConfig); !os.IsNotExist(err) {
		t.Fatalf("temporary run configuration remains: %v", err)
	}
}
