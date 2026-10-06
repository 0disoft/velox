package initializer

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/velox/internal/manifest"
)

func TestTemplateCatalogMatchesGeneratedPermissions(t *testing.T) {
	var names []string
	for _, template := range Templates() {
		names = append(names, template.Name)
		target := filepath.Join(t.TempDir(), "sample")
		if _, err := CreateFromTemplate(target, template.Name); err != nil {
			t.Fatal(err)
		}
		config, err := manifest.Load(filepath.Join(target, "velox.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(template.Permissions, config.Security.Permissions) {
			t.Fatalf("%s: catalog=%v generated=%v", template.Name, template.Permissions, config.Security.Permissions)
		}
		if template.Description == "" || template.InitCommand == "" {
			t.Fatalf("incomplete catalog entry: %+v", template)
		}
	}
	if !reflect.DeepEqual(names, []string{"basic", "text-editor", "folder-browser", "tray-app"}) {
		t.Fatalf("unexpected catalog order: %v", names)
	}
}

func TestTemplateCatalogReturnsIndependentValues(t *testing.T) {
	first := Templates()
	first[0].Name = "changed"
	first[1].Permissions[0] = "changed"
	second := Templates()
	if second[0].Name != "basic" || second[1].Permissions[0] != "file.open" {
		t.Fatalf("catalog was modified: %+v", second)
	}
}
