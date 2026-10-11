package installer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsShortcutRemovalBounds(t *testing.T) {
	for _, size := range []int{maxShortcutBytes, maxShortcutBytes + 1} {
		t.Run(map[bool]string{false: "at-limit", true: "over-limit"}[size > maxShortcutBytes], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.lnk")
			data := bytes.Repeat([]byte{'x'}, size)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(data)
			err := removeWindowsShortcut(path, hex.EncodeToString(hash[:]))
			if size <= maxShortcutBytes {
				if err != nil {
					t.Fatalf("bounded owned shortcut removal failed: %v", err)
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("owned shortcut remains: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("oversized shortcut was removed")
				}
				if current, err := os.ReadFile(path); err != nil || !bytes.Equal(current, data) {
					t.Fatalf("oversized shortcut was changed: %v", err)
				}
			}
		})
	}
}

func TestWindowsRemovalPreservesShortcutChangedAfterValidation(t *testing.T) {
	for _, mode := range []string{"same-size-edit", "replacement", "directory"} {
		t.Run(mode, func(t *testing.T) {
			source, setup, _ := fixture(t)
			root := t.TempDir()
			menu := filepath.Join(root, "menu")
			registryRoot := `Software\VeloxInstallerTests\` + filepath.Base(root) + `\`
			env := windowsEnvironment(filepath.Join(root, "apps"), menu, registryRoot)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			keyPath := registryRoot + installed.AppID
			t.Cleanup(func() {
				_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
				_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
				_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
			})
			record, err := readState(installed.Directory, installed.AppID)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(menu, installed.AppID+".lnk")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := bytes.Repeat([]byte{'x'}, len(original))
			err = unregisterWindows(record, menu, registryRoot, func(record Record) error {
				if err := env.validateRemoval(record); err != nil {
					return err
				}
				if mode != "same-size-edit" {
					if err := os.Remove(path); err != nil {
						return err
					}
				}
				if mode == "directory" {
					return os.Mkdir(path, 0o700)
				}
				return os.WriteFile(path, changed, 0o600)
			})
			if err == nil {
				t.Fatal("unregistration deleted a changed shortcut after successful validation")
			}
			if mode == "directory" {
				info, err := os.Stat(path)
				if err != nil || !info.IsDir() {
					t.Fatalf("replacement directory was removed: %v", err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else if data, err := os.ReadFile(path); err != nil || !bytes.Equal(data, changed) {
				t.Fatalf("changed shortcut content was removed: %v", err)
			}
			key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
			if err != nil {
				t.Fatalf("shortcut failure removed registration: %v", err)
			}
			validationErr := validateWindowsRegistration(key, record)
			key.Close()
			if validationErr != nil {
				t.Fatalf("registration changed on shortcut failure: %v", validationErr)
			}
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := uninstall(record.App.ID, env); err != nil {
				t.Fatalf("retry with original shortcut failed: %v", err)
			}
		})
	}
}
