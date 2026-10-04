package manifest

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestLoadWindowResizable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox.json")
	for _, test := range []struct {
		fields                  string
		present, enabled, valid bool
	}{
		{`"width":480`, false, true, true},
		{`"resizable":true`, true, true, true},
		{`"resizable":false`, true, false, true},
		{`"resizable":"false"`, false, false, false},
		{`"resizable":0`, false, false, false},
		{`"resizable":{}`, false, false, false},
	} {
		writeTestFile(t, path, fmt.Sprintf(`{"schemaVersion":1,"app":{"id":"dev.velox.fixed","name":"Fixed","version":"1"},"window":{%s}}`, test.fields))
		got, err := Load(path)
		if (err == nil) != test.valid {
			t.Fatalf("%s: %v", test.fields, err)
		}
		if test.valid {
			if (got.Window.Resizable != nil) != test.present {
				t.Fatal("omission was not preserved")
			}
			enabled := got.Window.Resizable == nil || *got.Window.Resizable
			if enabled != test.enabled {
				t.Fatal(got.Window.Resizable)
			}
		}
	}
}
