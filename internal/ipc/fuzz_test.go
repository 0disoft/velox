package ipc

import (
	"encoding/json"
	"strings"
	"testing"
)

func FuzzDispatcher(f *testing.F) {
	for _, seed := range []string{
		`{"v":1,"id":1,"method":"app.getInfo","params":{}}`,
		`{"v":1,"id":2,"method":"window.getState","params":{}}`,
		`{"v":1,"id":3,"id":4,"method":"app.getInfo","params":{}}`,
		`{"v":1,"id":4,"method":"app.getInfo","params":{"nested":[[[null]]]}}`,
		`null`,
		`{`,
		strings.Repeat(" ", MaxRequestBytes+1),
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		dispatcher := NewDispatcher(
			Identity{ID: "dev.velox.fuzz", Name: "Fuzz", Version: "1.0.0", Platform: "windows"},
			[]string{PermissionAppInfo, PermissionWindow},
			&fakeWindow{state: "normal"},
		)
		response := dispatcher.Dispatch(json.RawMessage(data))
		if response.Version != Version {
			t.Fatalf("response version = %d", response.Version)
		}
		if response.OK == (response.Error != nil) {
			t.Fatalf("inconsistent response: %#v", response)
		}
		if _, err := json.Marshal(response); err != nil {
			t.Fatalf("response is not JSON serializable: %v", err)
		}
	})
}
