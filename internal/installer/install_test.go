package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/buildreport"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

func fixture(t *testing.T) (string, string, environment) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "portable")
	write := func(name string, data []byte) {
		t.Helper()
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("web/index.html", []byte("<h1>Example</h1>"))
	write("dev.velox.test.exe", []byte("test host"))
	assets, err := assettree.Scan(filepath.Join(source, "web"))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte("test host"))
	report := buildreport.Report{
		SchemaVersion: buildreport.SchemaVersion, ReleaseVersion: "test", Target: "windows-x64",
		App:         buildreport.App{ID: "dev.velox.test", Name: "Test App", Version: "1.0.0"},
		Contracts:   buildreport.Contracts{Manifest: 1, Runtime: 1, Host: 1, IPC: 1},
		Host:        buildreport.File{File: "dev.velox.test.exe", Bytes: 9, SHA256: hex.EncodeToString(hash[:])},
		Assets:      buildreport.Assets{Files: 1, Bytes: assets.TotalBytes, SHA256: assets.Digest},
		Permissions: []string{}, Outputs: buildreport.OutputCounts{PortableFiles: 4},
	}
	data, _ := json.Marshal(report)
	write("build-result.json", data)
	config := runtimeconfig.Config{RuntimeVersion: 1, App: runtimeconfig.App{ID: report.App.ID, Name: report.App.Name, Version: report.App.Version}, Assets: runtimeconfig.Assets{Root: "web", Entry: "index.html"}, Window: runtimeconfig.Window{Width: 800, Height: 600}, Security: runtimeconfig.Security{Permissions: []string{}}}
	data, _ = json.Marshal(config)
	write("velox.runtime.json", data)
	uninstaller := filepath.Join(root, "setup.exe")
	if err := os.WriteFile(uninstaller, []byte("test setup"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := environment{
		root:  filepath.Join(root, "installed"),
		check: func(Record) error { return nil },
		prepare: func(Record) (registration, error) {
			return registration{publish: func() error { return nil }, cleanup: func() {}}, nil
		},
		unregister:      func(Record) error { return nil },
		validateRemoval: func(Record) error { return nil },
		removable:       func(string) error { return nil },
	}
	return source, uninstaller, env
}

func TestInstallUninstallPreservesDataAndRefusesReplacement(t *testing.T) {
	source, setup, env := fixture(t)
	data := filepath.Join(filepath.Dir(env.root), "user-notes.md")
	if err := os.WriteFile(data, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := install(source, setup, env); err == nil {
		t.Fatal("replaced existing install")
	}
	if _, err := uninstall(result.AppID, env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("install remains: %v", err)
	}
	if contents, err := os.ReadFile(data); err != nil || string(contents) != "keep me" {
		t.Fatal("user data changed")
	}
}

func TestUninstallRefusesChangedAndForeignFiles(t *testing.T) {
	for _, name := range []string{"notes.md", "app/web/index.html"} {
		t.Run(name, func(t *testing.T) {
			source, setup, env := fixture(t)
			result, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(result.Directory, filepath.FromSlash(name))
			if err := os.WriteFile(path, []byte("user content"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := uninstall(result.AppID, env); err == nil {
				t.Fatal("removed foreign or changed file")
			}
			if _, err := os.Stat(filepath.Join(result.Directory, "uninstall.exe")); err != nil {
				t.Fatal("refusal partially removed installation")
			}
		})
	}
}

func TestInstallRollbackAndUninstallPreflight(t *testing.T) {
	source, setup, env := fixture(t)
	prepare := env.prepare
	env.prepare = func(record Record) (registration, error) {
		prepared, err := prepare(record)
		prepared.publish = func() error { return errors.New("injected registration failure") }
		return prepared, err
	}
	if _, err := install(source, setup, env); err == nil {
		t.Fatal("ignored integration failure")
	}
	entries, err := os.ReadDir(env.root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed installation residue: %v %v", entries, err)
	}
	env.prepare = prepare
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	env.validateRemoval = func(Record) error { return errors.New("foreign registration") }
	if _, err := uninstall(result.AppID, env); err == nil {
		t.Fatal("ignored foreign registration")
	}
	if _, err := os.Stat(filepath.Join(result.Directory, "app", result.AppID+".exe")); err != nil {
		t.Fatal("preflight deleted files")
	}
}

func TestInstallerRejectsTraversalAndInvalidOwnership(t *testing.T) {
	source, setup, env := fixture(t)
	for _, id := range []string{"../outside", "C:/outside", "dev.velox.test/child"} {
		if _, err := uninstall(id, env); err == nil {
			t.Fatal("accepted invalid ID")
		}
	}
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	record, err := readState(result.Directory, result.AppID)
	if err != nil {
		t.Fatal(err)
	}
	record.Files[0].Path = "../outside"
	if err := writeState(result.Directory, record); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstall(result.AppID, env); err == nil {
		t.Fatal("accepted forged path")
	}
}

func TestInstallRecordsShortcutBeforePublication(t *testing.T) {
	source, setup, env := fixture(t)
	hash := strings.Repeat("a", 64)
	published, cleaned := false, false
	env.prepare = func(record Record) (registration, error) {
		if _, err := os.Stat(record.Directory); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("installation visible before shortcut preparation: %v", err)
		}
		return registration{
			shortcutHash: hash,
			cleanup:      func() { cleaned = true },
			publish: func() error {
				stored, err := readState(record.Directory, record.App.ID)
				if err != nil {
					return err
				}
				if stored.ShortcutHash != hash {
					t.Fatalf("shortcut ownership not committed before publication: %q", stored.ShortcutHash)
				}
				published = true
				return nil
			},
		}, nil
	}
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	if !published || !cleaned {
		t.Fatalf("registration lifecycle incomplete: published=%v cleaned=%v", published, cleaned)
	}
	if _, err := uninstall(result.AppID, env); err != nil {
		t.Fatal(err)
	}
}

func TestInstallPreparationFailureLeavesNoPublishedInstallation(t *testing.T) {
	source, setup, env := fixture(t)
	env.prepare = func(Record) (registration, error) {
		return registration{}, errors.New("injected preparation failure")
	}
	if _, err := install(source, setup, env); err == nil {
		t.Fatal("ignored preparation failure")
	}
	entries, err := os.ReadDir(env.root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("preparation failure left installation files: %v %v", entries, err)
	}
}
