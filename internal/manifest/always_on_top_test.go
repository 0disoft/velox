package manifest

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestLoadAlwaysOnTop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox.json")
	for _, test := range []struct {
		fields         string
		enabled, valid bool
	}{
		{`"width":960`, false, true},
		{`"alwaysOnTop":false`, false, true},
		{`"alwaysOnTop":true`, true, true},
		{`"alwaysOnTop":"true"`, false, false},
		{`"alwaysOnTop":1`, false, false},
		{`"alwaysOnTop":{}`, false, false},
	} {
		writeTestFile(t, path, fmt.Sprintf(`{"schemaVersion":1,"app":{"id":"dev.velox.topmost","name":"Topmost","version":"1"},"window":{%s}}`, test.fields))
		got, err := Load(path)
		if (err == nil) != test.valid || (test.valid && got.Window.AlwaysOnTop != test.enabled) {
			t.Fatalf("%s: %+v, %v", test.fields, got.Window, err)
		}
	}
}
