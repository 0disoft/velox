package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/manifest"
)

func TestResizableManifestRoundTrip(t *testing.T) {
	for _, setting := range []*bool{nil, new(true), new(false)} {
		value := manifest.Resolved{Manifest: manifest.Manifest{
			App:    manifest.App{ID: "dev.velox.fixed", Name: "Fixed", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"},
			Window: manifest.Window{Width: 480, Height: 360, Resizable: setting},
		}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(body)
		if err != nil {
			t.Fatal(err)
		}
		if (got.Window.Resizable == nil) != (setting == nil) {
			t.Fatal("omission changed")
		}
		if setting != nil && *got.Window.Resizable != *setting {
			t.Fatal("value changed")
		}
		if strings.Contains(string(body), `"resizable"`) != (setting != nil) {
			t.Fatal(string(body))
		}
		if setting != nil && *setting {
			for _, invalid := range []string{`1`, `"true"`, `{}`} {
				if _, err := Parse([]byte(strings.Replace(string(body), `"resizable":true`, `"resizable":`+invalid, 1))); err == nil {
					t.Fatal("invalid resizable accepted")
				}
			}
		}
	}
}
