package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/activationkey"
	"github.com/0disoft/velox/internal/manifest"
)

func TestActivationShortcutRoundTrip(t *testing.T) {
	for _, shortcut := range []activationkey.Shortcut{"", "Ctrl+Alt+V", "Ctrl+Alt+Shift+9"} {
		value := manifest.Resolved{Manifest: manifest.Manifest{
			App:    manifest.App{ID: "dev.velox.shortcut-test", Name: "Shortcut test", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"}, Window: manifest.Window{Width: 640, Height: 480, ActivationShortcut: shortcut},
			Security: manifest.Security{Permissions: []string{}},
		}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(body)
		if err != nil || got.Window.ActivationShortcut != shortcut || (strings.Contains(string(body), `"activationShortcut"`) != (shortcut != "")) {
			t.Fatal(string(body), got, err)
		}
		if shortcut != "" {
			for _, invalid := range []string{`null`, `false`, `"Ctrl+V"`, `"Ctrl+Alt+F12"`} {
				raw := strings.Replace(string(body), `"`+string(shortcut)+`"`, invalid, 1)
				if _, err := Parse([]byte(raw)); err == nil {
					t.Fatal("accepted", invalid)
				}
			}
		}
	}
}
