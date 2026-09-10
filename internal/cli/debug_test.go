package cli

import (
	"bytes"
	"io"
	"testing"
)

func TestRunDebugIsExplicitAndForwarded(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		_, config, host := cliFixture(t)
		args := []string{"run", "--config", config, "--json"}
		if enabled {
			args = append(args, "--debug")
		}
		var output bytes.Buffer
		called := false
		code := Run(args, Dependencies{
			HostPath: host, Stdout: &output, Stderr: io.Discard,
			HostLauncher: func(_, _ string, debug bool, _, _ io.Writer) (int, error) {
				called = true
				if debug != enabled {
					t.Fatalf("debug = %t, want %t", debug, enabled)
				}
				return 0, nil
			},
		})
		if code != 0 || !called {
			t.Fatalf("run failed: exit=%d called=%t output=%s", code, called, &output)
		}
	}
}
