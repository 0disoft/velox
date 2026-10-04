package initializer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/manifest"
	bridgetypes "github.com/0disoft/velox/types"
)

func TestCreateWritesDependencyFreeProject(t *testing.T) {
	target := filepath.Join(t.TempDir(), "my-app")
	result, err := Create(target)
	if err != nil {
		t.Fatal(err)
	}
	if result.AppID != "dev.velox.my-app" || result.AppName != "My App" {
		t.Fatalf("unexpected identity: %+v", result)
	}
	for _, relative := range result.Files {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 3 {
		t.Fatalf("unexpected project root: entries=%v err=%v", entries, err)
	}
	if want := []string{"velox.json", "velox.d.ts", "web/index.html", "web/style.css", "web/app.js"}; !reflect.DeepEqual(result.Files, want) {
		t.Fatalf("unexpected file inventory: %v", result.Files)
	}
	types, err := os.ReadFile(filepath.Join(target, "velox.d.ts"))
	if err != nil || string(types) != bridgetypes.Declaration() {
		t.Fatalf("declaration differs from CLI embed: %v", err)
	}
	config, err := manifest.Load(filepath.Join(target, "velox.json"))
	if err != nil || len(config.Security.Permissions) != 0 {
		t.Fatalf("generated manifest or default permissions changed: %v", err)
	}
	assets, err := assettree.Scan(filepath.Join(target, "web"))
	if err != nil || len(assets.Files) != 3 {
		t.Fatalf("unexpected web assets: %+v %v", assets, err)
	}
	app, err := os.ReadFile(filepath.Join(target, "web", "app.js"))
	if err != nil || !strings.HasPrefix(string(app), "/// <reference path=\"../velox.d.ts\" />\n") {
		t.Fatalf("missing JavaScript editor reference: %v", err)
	}
}

func TestCreateRefusesDeclarationConflictWithoutPartialWrites(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, "velox.d.ts")
	if err := os.WriteFile(path, []byte("user declaration"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(target); err == nil {
		t.Fatal("expected declaration conflict")
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 1 || entries[0].Name() != "velox.d.ts" {
		t.Fatalf("partial files remained: %v %v", entries, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "user declaration" {
		t.Fatalf("user declaration changed: %q %v", data, err)
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

func TestCreateRejectsLinkedWebDirectoryWithoutExternalWrites(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "project")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(target, "web")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := Create(target); err == nil {
		t.Fatal("Create() accepted a linked web directory")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("Create() wrote outside the project: %v", entries)
	}
}
