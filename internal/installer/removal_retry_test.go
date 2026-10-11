package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLateUserFileKeepsRemovalRetryRecord(t *testing.T) {
	source, setup, env := fixture(t)
	installed, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(installed.Directory, "app", "web", "late-notes.md")
	env.unregister = func(Record) error {
		return os.WriteFile(note, []byte("late user content\r\n"), 0o600)
	}
	if _, err := uninstall(installed.AppID, env); err == nil {
		t.Fatal("removal succeeded with an additional file")
	}
	if data, err := os.ReadFile(note); err != nil || string(data) != "late user content\r\n" {
		t.Fatalf("late user file changed: %v", err)
	}
	if _, err := readState(installed.Directory, installed.AppID); err != nil {
		t.Fatalf("late directory failure lost its retry record: %v", err)
	}
	if err := os.Remove(note); err != nil {
		t.Fatal(err)
	}
	env.unregister = func(Record) error { return nil }
	if _, err := uninstall(installed.AppID, env); err != nil {
		t.Fatalf("cannot retry after moving the conflicting file: %v", err)
	}
}

func TestLateOwnershipChangeIsPreserved(t *testing.T) {
	source, setup, env := fixture(t)
	installed, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(installed.Directory, stateFile)
	var changed []byte
	env.unregister = func(record Record) error {
		record.ShortcutHash = strings.Repeat("a", 64)
		changed, err = json.Marshal(record)
		if err != nil {
			return err
		}
		return os.WriteFile(path, changed, 0o600)
	}
	if _, err := uninstall(installed.AppID, env); err == nil {
		t.Fatal("late ownership change was ignored")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != string(changed) {
		t.Fatalf("changed ownership record was overwritten or removed: %v", err)
	}
}

func TestStateRestorationDoesNotOverwrite(t *testing.T) {
	source, setup, env := fixture(t)
	installed, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	record, err := readState(installed.Directory, installed.AppID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(installed.Directory, stateFile)
	if err := os.WriteFile(path, []byte("new user file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := restoreState(installed.Directory, record); err == nil {
		t.Fatal("restoration replaced a file at the ownership path")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "new user file" {
		t.Fatalf("restoration overwrote user content: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(installed.Directory, ".velox-state-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("failed restoration left temporary records: %v %v", matches, err)
	}
}
