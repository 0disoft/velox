package installer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicationFailurePreservesUserFiles(t *testing.T) {
	for _, relative := range []string{"notes.md", "app/new-folder/notes.md", "app/web/index.html"} {
		t.Run(relative, func(t *testing.T) {
			source, setup, env := fixture(t)
			prepare := env.prepare
			var path string
			var original []byte
			failure := errors.New("publication failed")
			env.prepare = func(record Record) (registration, error) {
				prepared, err := prepare(record)
				prepared.publish = func() error {
					path = filepath.Join(record.Directory, filepath.FromSlash(relative))
					original, err = os.ReadFile(path)
					if err != nil && !errors.Is(err, os.ErrNotExist) {
						t.Fatal(err)
					}
					if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte("user content\r\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					return failure
				}
				return prepared, err
			}
			if _, err := install(source, setup, env); !errors.Is(err, failure) {
				t.Fatalf("publication failure was lost: %v", err)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "user content\r\n" {
				t.Fatalf("rollback deleted or changed user content: %v", err)
			}
			target := filepath.Join(env.root, "dev.velox.test")
			if _, err := readState(target, "dev.velox.test"); err != nil {
				t.Fatalf("rollback lost its retry record: %v", err)
			}
			if original == nil {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := uninstall("dev.velox.test", env); err != nil {
				t.Fatalf("cannot remove retained installation after resolving conflict: %v", err)
			}
			env.prepare = prepare
			if _, err := install(source, setup, env); err != nil {
				t.Fatalf("cannot retry installation: %v", err)
			}
		})
	}
}

func TestPublicationFailurePreservesChangedOwnership(t *testing.T) {
	source, setup, env := fixture(t)
	prepare := env.prepare
	var path string
	var changed []byte
	env.prepare = func(record Record) (registration, error) {
		prepared, err := prepare(record)
		prepared.publish = func() error {
			path = filepath.Join(record.Directory, stateFile)
			stored, err := readState(record.Directory, record.App.ID)
			if err != nil {
				t.Fatal(err)
			}
			stored.ShortcutHash = strings.Repeat("a", 64)
			changed, err = json.Marshal(stored)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, changed, 0o600); err != nil {
				t.Fatal(err)
			}
			return errors.New("publication failed")
		}
		return prepared, err
	}
	if _, err := install(source, setup, env); err == nil {
		t.Fatal("publication unexpectedly succeeded")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != string(changed) {
		t.Fatalf("rollback removed a changed ownership record: %v", err)
	}
}
