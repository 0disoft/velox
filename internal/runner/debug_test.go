package runner

import (
	"reflect"
	"testing"
)

func TestHostDebugArgumentsDoNotChangeConfigOrEnvironment(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		command := hostCommand("host.exe", "project with spaces/runtime.json", enabled)
		want := []string{"host.exe", "--config", "project with spaces/runtime.json"}
		if enabled {
			want = append(want, "--debug")
		}
		if !reflect.DeepEqual(command.Args, want) || command.Env != nil {
			t.Fatalf("unexpected host command: args=%q env=%v", command.Args, command.Env)
		}
	}
}
