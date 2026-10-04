package manifest

import (
	"path/filepath"
	"testing"
)

func TestActivationShortcutManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox.json")
	for _, test := range []struct {
		field, want string
		invalid     bool
	}{
		{"", "", false}, {`,"activationShortcut":""`, "", false},
		{`,"activationShortcut":"Ctrl+Alt+V"`, "Ctrl+Alt+V", false},
		{`,"activationShortcut":"Ctrl+Alt+Shift+9"`, "Ctrl+Alt+Shift+9", false},
		{`,"activationShortcut":null`, "", true}, {`,"activationShortcut":true`, "", true},
		{`,"activationShortcut":"Win+V"`, "", true}, {`,"activationShortcut":"Ctrl+Alt+v"`, "", true},
	} {
		writeTestFile(t, path, `{"schemaVersion":1,"app":{"id":"dev.velox.shortcut-test","name":"Shortcut test","version":"1"},"window":{"width":640,"height":480`+test.field+`}}`)
		got, err := Load(path)
		if (err != nil) != test.invalid || (!test.invalid && string(got.Window.ActivationShortcut) != test.want) {
			t.Fatal(test, got.Window, err)
		}
	}
}
