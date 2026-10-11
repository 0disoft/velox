package installer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsPartialRegistrationRecovery(t *testing.T) {
	for _, test := range []struct {
		name   string
		values func(Record) map[string]string
		child  bool
		refuse bool
	}{
		{"empty-key", func(Record) map[string]string { return nil }, false, false},
		{"identity-only", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory)}
		}, false, false},
		{"identity-and-owner", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory), "VeloxInstaller": stateSchema}
		}, false, false},
		{"legacy-complete", func(r Record) map[string]string {
			return map[string]string{"VeloxInstaller": stateSchema, "InstallLocation": r.Directory}
		}, false, false},
		{"legacy-partial", func(Record) map[string]string { return map[string]string{"VeloxInstaller": stateSchema} }, false, true},
		{"foreign-child", func(Record) map[string]string { return nil }, true, true},
		{"identity-foreign", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory + "-foreign"), "VeloxInstaller": stateSchema, "InstallLocation": r.Directory}
		}, false, true},
		{"identity-malformed", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": "not-json", "VeloxInstaller": stateSchema, "InstallLocation": r.Directory}
		}, false, true},
		{"location-changed", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory), "InstallLocation": r.Directory + "-foreign"}
		}, false, true},
		{"owner-changed", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory), "VeloxInstaller": "another-installer"}
		}, false, true},
		{"identity-with-child", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory)}
		}, true, true},
		{"foreign-value", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory), "UserNotes": "keep this registry data"}
		}, false, true},
		{"changed-command", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory), "UninstallString": "another application"}
		}, false, true},
		{"changed-display-name", func(r Record) map[string]string {
			return map[string]string{"VeloxInstallIdentity": testRegistrationIdentity(r.Directory), "DisplayName": "User edited name"}
		}, false, true},
		{"legacy-foreign-value", func(r Record) map[string]string {
			return map[string]string{"VeloxInstaller": stateSchema, "InstallLocation": r.Directory, "UserNotes": "keep this registry data"}
		}, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, setup, _ := fixture(t)
			root := t.TempDir()
			registryRoot := `Software\VeloxInstallerTests\` + filepath.Base(root) + `\`
			env := windowsEnvironment(filepath.Join(root, "apps"), filepath.Join(root, "menu"), registryRoot)
			path := registryRoot + "dev.velox.test"
			t.Cleanup(func() {
				_ = registry.DeleteKey(registry.CURRENT_USER, path+`\foreign`)
				_ = registry.DeleteKey(registry.CURRENT_USER, path)
				_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
				_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
			})
			prepare := env.prepare
			stopped := false
			env.prepare = func(record Record) (registration, error) {
				prepared, err := prepare(record)
				if err != nil {
					return registration{}, err
				}
				prepared.publish = func() error {
					key, existing, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
					if err != nil || existing {
						t.Fatalf("isolated key creation: existing=%v err=%v", existing, err)
					}
					defer key.Close()
					for name, value := range test.values(record) {
						if err := key.SetStringValue(name, value); err != nil {
							t.Fatal(err)
						}
					}
					if test.child {
						child, _, err := registry.CreateKey(registry.CURRENT_USER, path+`\foreign`, registry.SET_VALUE)
						if err != nil {
							t.Fatal(err)
						}
						child.Close()
					}
					stopped = true
					panic("injected interruption with partial registration")
				}
				return prepared, nil
			}
			func() {
				defer func() { _ = recover() }()
				_, _ = install(source, setup, env)
			}()
			if !stopped {
				t.Fatal("did not reach registration interruption")
			}
			_, err := uninstall("dev.velox.test", env)
			if test.refuse {
				if err == nil {
					t.Fatal("removed a changed or unrecognizable registration")
				}
				if _, err := os.Stat(filepath.Join(env.root, "dev.velox.test", "uninstall.exe")); err != nil {
					t.Fatalf("refusal partially deleted installation: %v", err)
				}
				record, err := readState(filepath.Join(env.root, "dev.velox.test"), "dev.velox.test")
				if err != nil {
					t.Fatal(err)
				}
				key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
				if err != nil {
					t.Fatalf("refusal removed the registration: %v", err)
				}
				defer key.Close()
				for name, expected := range test.values(record) {
					actual, _, err := key.GetStringValue(name)
					if err != nil || actual != expected {
						t.Fatalf("refusal changed %s: %v", name, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("interrupted registration cannot be removed: %v", err)
			}
			if _, err := os.Stat(filepath.Join(env.root, "dev.velox.test")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("installed tree remains: %v", err)
			}
			key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
			if err == nil {
				key.Close()
				t.Fatal("partial registry key remains")
			}
			if !errors.Is(err, registry.ErrNotExist) {
				t.Fatal(err)
			}
		})
	}
}

func testRegistrationIdentity(directory string) string {
	data, _ := json.Marshal(map[string]string{"schemaVersion": stateSchema, "directory": directory})
	return string(data)
}
