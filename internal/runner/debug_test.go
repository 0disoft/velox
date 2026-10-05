package runner

import (
	"reflect"
	"testing"
)

func TestHostDebugArgumentsDoNotChangeConfigOrEnvironment(t *testing.T) {
	for _, options := range []Options{{}, {Debug: true}, {Watch: true}, {Debug: true, Watch: true}} {
		command := hostCommand("host.exe", "project with spaces/runtime.json", options)
		want := []string{"host.exe", "--config", "project with spaces/runtime.json"}
		if options.Debug {
			want = append(want, "--debug")
		}
		if options.Watch {
			want = append(want, "--watch")
		}
		if !reflect.DeepEqual(command.Args, want) || command.Env != nil {
			t.Fatalf("unexpected host command: args=%q env=%v", command.Args, command.Env)
		}
	}
}
