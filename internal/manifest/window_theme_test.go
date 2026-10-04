package manifest

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestLoadFollowSystemTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox.json")
	for _, test := range []struct {
		fields         string
		enabled, valid bool
	}{
		{`"width":480`, false, true}, {`"followSystemTheme":false`, false, true},
		{`"followSystemTheme":true`, true, true}, {`"followSystemTheme":"true"`, false, false},
		{`"followSystemTheme":1`, false, false}, {`"followSystemTheme":{}`, false, false},
	} {
		writeTestFile(t, path, fmt.Sprintf(`{"schemaVersion":1,"app":{"id":"dev.velox.theme","name":"Theme","version":"1"},"window":{%s}}`, test.fields))
		got, err := Load(path)
		if (err == nil) != test.valid || (test.valid && got.Window.FollowSystemTheme != test.enabled) {
			t.Fatalf("%s: %+v, %v", test.fields, got.Window, err)
		}
	}
}
