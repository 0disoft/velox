package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRemovalJournalResumesMissingRecord(t *testing.T) {
	for _, phase := range []string{"record-removed", "root-removed"} {
		t.Run(phase, func(t *testing.T) {
			source, setup, env := fixture(t)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			record, err := readState(installed.Directory, installed.AppID)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			journal := filepath.Join(env.root, ".velox-removal-"+installed.AppID+".json")
			if err := os.WriteFile(journal, data, 0o600); err != nil {
				t.Fatal(err)
			}
			for _, file := range record.Files {
				if err := os.Remove(filepath.Join(installed.Directory, filepath.FromSlash(file.Path))); err != nil {
					t.Fatal(err)
				}
			}
			for _, relative := range []string{"app/web", "app", stateFile} {
				if err := os.Remove(filepath.Join(installed.Directory, filepath.FromSlash(relative))); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "root-removed" {
				if err := os.Remove(installed.Directory); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := uninstall(installed.AppID, env); err != nil {
				t.Fatalf("cannot resume final removal from its journal: %v", err)
			}
			for _, path := range []string{installed.Directory, journal} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("completed removal retained %q: %v", path, err)
				}
			}
		})
	}
}

func TestRemovalPreservesConflictingJournal(t *testing.T) {
	source, setup, env := fixture(t)
	installed, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(env.root, ".velox-removal-"+installed.AppID+".json")
	if err := os.WriteFile(journal, []byte("foreign metadata"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstall(installed.AppID, env); err == nil {
		t.Fatal("removed installation with a conflicting recovery record")
	}
	if _, err := os.Stat(filepath.Join(installed.Directory, "uninstall.exe")); err != nil {
		t.Fatalf("conflict caused partial removal: %v", err)
	}
	if data, err := os.ReadFile(journal); err != nil || string(data) != "foreign metadata" {
		t.Fatalf("conflicting journal changed: %v", err)
	}
}

func TestInstallPreservesPendingRemoval(t *testing.T) {
	source, setup, env := fixture(t)
	if err := os.MkdirAll(env.root, 0o700); err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(env.root, ".velox-removal-dev.velox.test.json")
	if err := os.WriteFile(journal, []byte("pending record"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := install(source, setup, env); err == nil {
		t.Fatal("new install bypassed unresolved removal metadata")
	}
	if data, err := os.ReadFile(journal); err != nil || string(data) != "pending record" {
		t.Fatalf("pending metadata changed: %v", err)
	}
}
