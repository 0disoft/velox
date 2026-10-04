package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/manifest"
)

func TestFollowSystemThemeRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		value := manifest.Resolved{Manifest: manifest.Manifest{
			App:    manifest.App{ID: "dev.velox.theme", Name: "Theme", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"},
			Window: manifest.Window{Width: 480, Height: 360, FollowSystemTheme: enabled},
		}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(body)
		if err != nil || got.Window.FollowSystemTheme != enabled {
			t.Fatal(got, err)
		}
		if strings.Contains(string(body), `"followSystemTheme"`) != enabled {
			t.Fatal(string(body))
		}
		if enabled {
			for _, invalid := range []string{`1`, `"true"`, `{}`} {
				if _, err := Parse([]byte(strings.Replace(string(body), `"followSystemTheme":true`, `"followSystemTheme":`+invalid, 1))); err == nil {
					t.Fatal("non-boolean theme accepted")
				}
			}
		}
	}
}
