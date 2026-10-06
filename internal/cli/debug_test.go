package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/velox/internal/runner"
)

func TestRunDebugIsExplicitAndForwarded(t *testing.T) {
	for _, options := range []runner.Options{{}, {Debug: true}, {Watch: true}, {Debug: true, Watch: true}} {
		_, config, host := cliFixture(t)
		args := []string{"run", "--config", config, "--json"}
		if options.Debug {
			args = append(args, "--debug")
		}
		if options.Watch {
			args = append(args, "--watch")
		}
		var output bytes.Buffer
		called := false
		code := Run(args, Dependencies{
			HostPath: host, Stdout: &output, Stderr: io.Discard,
			HostLauncher: func(_, _ string, got runner.Options, _, _ io.Writer) (int, error) {
				called = true
				if got != options {
					t.Fatalf("options = %+v, want %+v", got, options)
				}
				return 0, nil
			},
		})
		if code != 0 || !called {
			t.Fatalf("run failed: exit=%d called=%t output=%s", code, called, &output)
		}
	}
}

func TestRunDebugJSONKeepsStdoutJSONAndForwardsOnlyOptInStderr(t *testing.T) {
	for _, debug := range []bool{false, true} {
		_, config, host := cliFixture(t)
		args := []string{"run", "--config", config, "--json"}
		if debug {
			args = append(args, "--debug")
		}
		var stdout, stderr bytes.Buffer
		code := Run(args, Dependencies{
			HostPath: host, Stdout: &stdout, Stderr: &stderr,
			HostLauncher: func(_, _ string, _ runner.Options, childOut, childErr io.Writer) (int, error) {
				if childOut != io.Discard || (childErr == io.Discard) == debug {
					t.Fatal("incorrect child output forwarding")
				}
				_, _ = io.WriteString(childOut, "not JSON")
				_, _ = io.WriteString(childErr, "velox-debug: uncaught-error app.js:3:4\n")
				return 0, nil
			},
		})
		if code != 0 || !json.Valid(stdout.Bytes()) || (stderr.Len() > 0) != debug {
			t.Fatalf("debug=%t code=%d stdout=%q stderr=%q", debug, code, stdout.String(), stderr.String())
		}
	}
}

type watchJSONOutput struct {
	bytes.Buffer
	lines chan string
}

func (w *watchJSONOutput) Write(data []byte) (int, error) {
	n, err := w.Buffer.Write(data)
	w.lines <- string(data)
	return n, err
}

func TestRunWatchJSONSeparatesManifestNoticesFromHostDiagnostics(t *testing.T) {
	for _, debug := range []bool{false, true} {
		t.Run(map[bool]string{false: "debug-off", true: "debug-on"}[debug], func(t *testing.T) {
			_, config, host := cliFixture(t)
			valid, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"run", "--config", config, "--watch", "--json"}
			if debug {
				args = append(args, "--debug")
			}
			var stdout bytes.Buffer
			stderr := &watchJSONOutput{lines: make(chan string, 16)}
			awaitNotice := func(want string) {
				t.Helper()
				timer := time.NewTimer(5 * time.Second)
				defer timer.Stop()
				for {
					select {
					case line := <-stderr.lines:
						if strings.Contains(line, want) {
							return
						}
					case <-timer.C:
						t.Fatalf("timed out waiting for %q", want)
					}
				}
			}
			code := Run(args, Dependencies{
				HostPath: host, Stdout: &stdout, Stderr: stderr,
				HostLauncher: func(_, _ string, options runner.Options, childOut, childErr io.Writer) (int, error) {
					if !options.Watch || options.Debug != debug || childOut != io.Discard {
						t.Fatal("incorrect host options or stdout routing")
					}
					if !debug && childErr != io.Discard {
						t.Fatal("JSON watch exposed host stderr with debug off")
					}
					_, _ = io.WriteString(childOut, "private host stdout\n")
					_, _ = io.WriteString(childErr, "private host diagnostic\n")
					if err := os.WriteFile(config, []byte(`{`), 0o644); err != nil {
						t.Fatal(err)
					}
					awaitNotice("watch: manifest error:")
					if err := os.WriteFile(config, valid, 0o644); err != nil {
						t.Fatal(err)
					}
					awaitNotice("manifest changed; restart required")
					return 0, nil
				},
			})
			var envelope Envelope
			if code != 0 || json.Unmarshal(stdout.Bytes(), &envelope) != nil || !envelope.OK || envelope.Command != "run" {
				t.Fatalf("code=%d stdout=%q", code, stdout.String())
			}
			output := stderr.String()
			if !strings.Contains(output, "watch: manifest error:") || !strings.Contains(output, "restart required") || strings.Contains(output, "private host diagnostic") != debug {
				t.Fatalf("debug=%t incorrect stderr=%q", debug, output)
			}
		})
	}
}
