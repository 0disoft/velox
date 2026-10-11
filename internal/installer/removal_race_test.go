package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemovalPreservesFileChangedAfterPreflight(t *testing.T) {
	for _, mode := range []string{"same-size-edit", "replacement", "directory"} {
		t.Run(mode, func(t *testing.T) {
			source, setup, env := fixture(t)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(installed.Directory, "app", "web", "index.html")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := []byte("<h1>My note</h1>")
			if len(changed) != len(original) {
				t.Fatal("fixture must preserve file size")
			}
			mutated := false
			env.removable = func(string) error {
				if mutated {
					return nil
				}
				mutated = true
				switch mode {
				case "same-size-edit":
					return os.WriteFile(path, changed, 0o600)
				case "replacement":
					if err := os.Remove(path); err != nil {
						return err
					}
					return os.WriteFile(path, changed, 0o600)
				default:
					if err := os.Remove(path); err != nil {
						return err
					}
					return os.Mkdir(path, 0o700)
				}
			}
			unregistered := false
			env.unregister = func(Record) error { unregistered = true; return nil }
			if _, err := uninstall(installed.AppID, env); err == nil {
				t.Fatal("uninstall deleted a changed file after successful preflight")
			}
			if unregistered {
				t.Fatal("failed file removal unregistered the installation")
			}
			if _, err := readState(installed.Directory, installed.AppID); err != nil {
				t.Fatalf("retry ownership was lost: %v", err)
			}
			if mode == "directory" {
				info, err := os.Stat(path)
				if err != nil || !info.IsDir() {
					t.Fatalf("replacement directory was removed: %v", err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else if data, err := os.ReadFile(path); err != nil || string(data) != string(changed) {
				t.Fatalf("changed user content was deleted: %v", err)
			}
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			env.removable = func(string) error { return nil }
			if _, err := uninstall(installed.AppID, env); err != nil {
				t.Fatalf("retry after resolving the conflict failed: %v", err)
			}
		})
	}
}
