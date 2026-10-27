package installer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsRollbackPreservesForeignPublication(t *testing.T) {
	for _, kind := range []string{"registration", "shortcut"} {
		t.Run(kind, func(t *testing.T) {
			source, setup, _ := fixture(t)
			root := t.TempDir()
			registryRoot := testRegistrationNamespace(root) + `\`
			keyPath := registryRoot + "dev.velox.test"
			linkPath := filepath.Join(root, "menu", "dev.velox.test.lnk")
			t.Cleanup(func() {
				_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
				_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
				_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
			})
			env := windowsEnvironment(filepath.Join(root, "apps"), filepath.Join(root, "menu"), registryRoot)
			unregistered := false
			unregister := env.unregister
			env.unregister = func(r Record) error { unregistered = true; return unregister(r) }
			prepare := env.prepare
			env.prepare = func(record Record) (registration, error) {
				prepared, err := prepare(record)
				publish := prepared.publish
				prepared.publish = func() error {
					if kind == "registration" {
						key, existing, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
						if err != nil || existing {
							t.Fatalf("fixture key: existing=%v %v", existing, err)
						}
						defer key.Close()
						if err := key.SetStringValue("ForeignOwner", "keep me"); err != nil {
							t.Fatal(err)
						}
					} else if err := os.WriteFile(linkPath, []byte("foreign shortcut"), 0o600); err != nil {
						t.Fatal(err)
					}
					return publish()
				}
				return prepared, err
			}
			if _, err := install(source, setup, env); err == nil {
				t.Fatal("foreign publication unexpectedly succeeded")
			}
			if unregistered {
				t.Fatal("rollback invoked ordinary integration removal")
			}
			if _, err := os.Stat(filepath.Join(env.root, "dev.velox.test")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unchanged owned files were not rolled back: %v", err)
			}
			if kind == "registration" {
				key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
				if err != nil {
					t.Fatal(err)
				}
				defer key.Close()
				value, _, err := key.GetStringValue("ForeignOwner")
				if err != nil || value != "keep me" {
					t.Fatalf("foreign registration changed: %q %v", value, err)
				}
			} else if data, err := os.ReadFile(linkPath); err != nil || string(data) != "foreign shortcut" {
				t.Fatalf("foreign shortcut changed: %v", err)
			}
		})
	}
}
