package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/manifest"
)

func TestAlwaysOnTopManifestRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		value := manifest.Resolved{Manifest: manifest.Manifest{
			App:    manifest.App{ID: "dev.velox.topmost", Name: "Topmost", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"},
			Window: manifest.Window{Width: 960, Height: 640, AlwaysOnTop: enabled},
		}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(body)
		if err != nil || got.Window.AlwaysOnTop != enabled {
			t.Fatal(got, err)
		}
		if strings.Contains(string(body), `"alwaysOnTop"`) != enabled {
			t.Fatalf("optional field = %s", body)
		}
		if enabled {
			for _, invalid := range []string{`1`, `"true"`, `{}`} {
				if _, err := Parse([]byte(strings.Replace(string(body), `"alwaysOnTop":true`, `"alwaysOnTop":`+invalid, 1))); err == nil {
					t.Fatal("non-boolean topmost accepted")
				}
			}
		}
	}
}
