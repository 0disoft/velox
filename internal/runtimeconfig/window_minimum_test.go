package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/manifest"
)

func TestMinimumWindowManifestRoundTrip(t *testing.T) {
	for _, dimensions := range [][2]uint{{0, 0}, {640, 480}, {640, 0}, {0, 480}} {
		value := manifest.Resolved{Manifest: manifest.Manifest{
			App:    manifest.App{ID: "dev.velox.minimum", Name: "Minimum", Version: "1"},
			Assets: manifest.Assets{Entry: "index.html"},
			Window: manifest.Window{Width: 960, Height: 640, MinWidth: dimensions[0], MinHeight: dimensions[1]},
		}}
		body, err := json.Marshal(FromManifest(value, "web"))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(body)
		if err != nil || parsed.Window.MinWidth != dimensions[0] || parsed.Window.MinHeight != dimensions[1] {
			t.Fatal(parsed, err)
		}
		if strings.Contains(string(body), `"minWidth"`) != (dimensions[0] != 0) || strings.Contains(string(body), `"minHeight"`) != (dimensions[1] != 0) {
			t.Fatalf("optional fields = %s", body)
		}
		for _, invalid := range []string{strings.Replace(string(body), `"width":960`, `"width":320,"minWidth":961`, 1),
			strings.Replace(string(body), `"height":640`, `"height":240,"minHeight":641`, 1)} {
			if _, err := Parse([]byte(invalid)); err == nil {
				t.Fatal("invalid runtime minimum accepted")
			}
		}
	}
}
