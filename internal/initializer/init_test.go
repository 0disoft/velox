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

func TestCreateTextEditorTemplate(t *testing.T) {
	target := filepath.Join(t.TempDir(), "my-editor")
	result, err := CreateFromTemplate(target, "text-editor")
	if err != nil {
		t.Fatal(err)
	}
	config, err := manifest.Load(filepath.Join(target, "velox.json"))
	if err != nil || !reflect.DeepEqual(config.Security.Permissions, []string{"file.open", "file.save"}) {
		t.Fatalf("unexpected permissions: %+v %v", config.Security.Permissions, err)
	}
	assets, err := assettree.Scan(filepath.Join(target, "web"))
	if err != nil || len(assets.Files) != 17 || len(result.Files) != 19 {
		t.Fatalf("unexpected inventory: %+v %+v %v", result.Files, assets, err)
	}
	for _, relative := range result.Files {
		data, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(relative)))
		if err != nil || len(data) == 0 {
			t.Fatalf("empty or missing %s: %v", relative, err)
		}
	}
	index, err := os.ReadFile(filepath.Join(target, "web", "index.html"))
	if err != nil || strings.Contains(string(index), "{{APP_NAME}}") || !strings.Contains(string(index), "My Editor") {
		t.Fatalf("app name not substituted: %v", err)
	}
	if !strings.Contains(string(index), `src="drafts.js"`) || !strings.Contains(string(index), `id="recovery-dialog"`) {
		t.Fatal("draft storage or recovery dialog is not delivered")
	}
	if !strings.Contains(string(index), `src="find.js"`) || !strings.Contains(string(index), `id="find-case"`) {
		t.Fatal("find module or match-case control is not delivered")
	}
	if !strings.Contains(string(index), `id="replace-row"`) || !strings.Contains(string(index), `id="replace-undo"`) {
		t.Fatal("replace controls are not delivered")
	}
}

func TestCreateFolderBrowserTemplate(t *testing.T) {
	target := filepath.Join(t.TempDir(), "my-browser")
	result, err := CreateFromTemplate(target, "folder-browser")
	if err != nil {
		t.Fatal(err)
	}
	config, err := manifest.Load(filepath.Join(target, "velox.json"))
	if err != nil || !reflect.DeepEqual(config.Security.Permissions, []string{"folder.read", "folder.readText"}) {
		t.Fatalf("unexpected permissions: %+v %v", config.Security.Permissions, err)
	}
	assets, err := assettree.Scan(filepath.Join(target, "web"))
	if err != nil || len(assets.Files) != 7 || len(result.Files) != 9 {
		t.Fatalf("unexpected inventory: %+v %+v %v", result.Files, assets, err)
	}
	for _, relative := range result.Files {
		data, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(relative)))
		if err != nil || len(data) == 0 || (relative != "velox.d.ts" && strings.Contains(string(data), "clipboard")) {
			t.Fatalf("missing asset or extra capability in %s: %v", relative, err)
		}
	}
	files, err := templateFiles("folder-browser", `<img src=x>`)
	if err != nil || strings.Contains(string(files[0].data), `<img src=x>`) || !strings.Contains(string(files[0].data), "&lt;img src=x&gt;") {
		t.Fatalf("unsafe app name: %v", err)
	}
	conflict := t.TempDir()
	if err := os.Mkdir(filepath.Join(conflict, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(conflict, "web", "refresh-cw.svg")
	if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFromTemplate(conflict, "folder-browser"); err == nil {
		t.Fatal("expected icon conflict")
	}
	if _, err := os.Stat(filepath.Join(conflict, "velox.json")); !os.IsNotExist(err) {
		t.Fatal("partial manifest remained")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("user icon changed: %q %v", data, err)
	}
}

func TestCreateTrayAppTemplate(t *testing.T) {
	target := filepath.Join(t.TempDir(), "my-tray")
	result, err := CreateFromTemplate(target, "tray-app")
	if err != nil {
		t.Fatal(err)
	}
	config, err := manifest.Load(filepath.Join(target, "velox.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Security.Permissions, []string{"notification.show"}) || !config.App.SingleInstance ||
		!config.Window.Tray || !config.Window.RememberState || !config.Window.FollowSystemTheme || config.Window.ActivationShortcut != "" {
		t.Fatalf("unexpected tray defaults: %+v", config.Manifest)
	}
	if config.Window.Width != 620 || config.Window.Height != 480 || config.Window.MinWidth != 360 || config.Window.MinHeight != 400 {
		t.Fatalf("unexpected window dimensions: %+v", config.Window)
	}
	want := []string{"velox.json", "velox.d.ts", "web/index.html", "web/style.css", "web/app.js", "web/bell.svg", "web/icons-license.txt"}
	if !reflect.DeepEqual(result.Files, want) {
		t.Fatalf("unexpected file inventory: %v", result.Files)
	}
	assets, err := assettree.Scan(filepath.Join(target, "web"))
	if err != nil || len(assets.Files) != 5 {
		t.Fatalf("unexpected web assets: %+v %v", assets, err)
	}
	files, err := templateFiles("tray-app", `<img src=x>`)
	if err != nil || strings.Contains(string(files[0].data), `<img src=x>`) || !strings.Contains(string(files[0].data), "&lt;img src=x&gt;") {
		t.Fatalf("unsafe app name: %v", err)
	}
	conflict := t.TempDir()
	if err := os.Mkdir(filepath.Join(conflict, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	icon := filepath.Join(conflict, "web", "bell.svg")
	if err := os.WriteFile(icon, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFromTemplate(conflict, "tray-app"); err == nil {
		t.Fatal("expected icon conflict")
	}
	if _, err := os.Stat(filepath.Join(conflict, "velox.json")); !os.IsNotExist(err) {
		t.Fatal("partial manifest remained")
	}
	data, err := os.ReadFile(icon)
	if err != nil || string(data) != "keep" {
		t.Fatalf("user icon changed: %q %v", data, err)
	}
}

func TestTextEditorEscapesNameAndPreservesConflictingIcon(t *testing.T) {
	files, err := templateFiles("text-editor", `<script>alert("name")</script>`)
	if err != nil || strings.Contains(string(files[0].data), `<script>alert`) || !strings.Contains(string(files[0].data), "&lt;script&gt;") {
		t.Fatalf("unsafe app name: %v", err)
	}
	target := t.TempDir()
	if err := os.Mkdir(filepath.Join(target, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "web", "save.svg")
	if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFromTemplate(target, "text-editor"); err == nil {
		t.Fatal("expected icon conflict")
	}
	if _, err := os.Stat(filepath.Join(target, "velox.json")); !os.IsNotExist(err) {
		t.Fatal("partial manifest remained")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("user icon changed: %q %v", data, err)
	}
}

func TestUnknownTemplateWritesNothing(t *testing.T) {
	for _, template := range []string{"", "unknown", "TEXT-EDITOR", "../text-editor"} {
		target := filepath.Join(t.TempDir(), "new-editor")
		if _, err := CreateFromTemplate(target, template); err != ErrUnknownTemplate {
			t.Fatalf("template %q: %v", template, err)
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("unknown template created target: %v", err)
		}
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
