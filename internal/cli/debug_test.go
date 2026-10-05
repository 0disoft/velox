package cli

import (
	"bytes"
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
