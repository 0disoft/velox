package builder

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/manifest"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

func TestBuildPreservesActivationShortcut(t *testing.T) {
	root, path, host := fixture(t)
	value, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	value.Window.ActivationShortcut = "Ctrl+Alt+Shift+V"
	value.Security.Permissions = []string{}
	body, err := json.Marshal(value.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, path, body)
	plan, err := buildplan.CreateBuild(buildplan.Options{ManifestPath: path, HostPath: host, OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	got, err := runtimeconfig.Load(filepath.Join(result.DirectoryPath, "velox.runtime.json"))
	if err != nil || got.Window.ActivationShortcut != value.Window.ActivationShortcut || got.Window.Tray || len(got.Security.Permissions) != 0 {
		t.Fatal(got, err)
	}
}
