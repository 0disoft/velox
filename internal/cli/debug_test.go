package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

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
