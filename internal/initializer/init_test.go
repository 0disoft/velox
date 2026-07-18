package initializer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCreateWritesDependencyFreeProject(t *testing.T) {
	target := filepath.Join(t.TempDir(), "my-app")
	result, err := Create(target)
	if err != nil {
		t.Fatal(err)
	}
	if result.AppID != "dev.actutum.my-app" || result.AppName != "My App" {
		t.Fatalf("unexpected identity: %+v", result)
	}
	for _, relative := range result.Files {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 2 {
		t.Fatalf("unexpected project root: entries=%v err=%v", entries, err)
	}
}

func TestCreateRefusesConflictWithoutPartialWrites(t *testing.T) {
	target := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(filepath.Join(target, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "web", "style.css"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(target); err == nil {
		t.Fatal("expected conflict")
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if names := []string{entries[0].Name()}; !reflect.DeepEqual(names, []string{"web"}) {
		t.Fatalf("partial files remained: %v", names)
	}
	data, err := os.ReadFile(filepath.Join(target, "web", "style.css"))
	if err != nil || string(data) != "keep" {
		t.Fatalf("conflicting file changed: %q %v", data, err)
	}
}
