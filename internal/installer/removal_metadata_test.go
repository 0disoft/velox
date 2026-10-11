package installer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnedRecordRemovalRejectsChangeAfterSnapshot(t *testing.T) {
	for _, journal := range []bool{false, true} {
		t.Run(map[bool]string{false: "internal-record", true: "recovery-journal"}[journal], func(t *testing.T) {
			source, setup, env := fixture(t)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			record, err := readState(installed.Directory, installed.AppID)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(record.Directory, stateFile)
			if journal {
				if err := prepareRemovalJournal(record); err != nil {
					t.Fatal(err)
				}
				path = removalJournalPath(record.Directory, record.App.ID)
			}
			snapshot, err := ownedRecordSnapshot(record, path)
			if err != nil {
				t.Fatal(err)
			}
			foreign := record
			foreign.App.Name = "Different owner"
			data, err := json.Marshal(foreign)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := removeRecordedFile(path, snapshot); err == nil {
				t.Fatal("removed metadata changed after its verified snapshot")
			}
			if err := removeOwnedRecord(record, path); err == nil {
				t.Fatal("removed current conflicting ownership metadata")
			}
			if current, err := os.ReadFile(path); err != nil || string(current) != string(data) {
				t.Fatalf("changed metadata was removed: %v", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := removeOwnedRecord(record, path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing metadata no longer supports retry: %v", err)
			}
		})
	}
}

func TestFinalRemovalPreservesReplacedOwnership(t *testing.T) {
	for _, mode := range []string{"user-note", "changed-record", "directory"} {
		t.Run(mode, func(t *testing.T) {
			source, setup, env := fixture(t)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			record, err := readState(installed.Directory, installed.AppID)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range record.Files {
				if err := os.Remove(filepath.Join(record.Directory, filepath.FromSlash(file.Path))); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(record.Directory, stateFile)
			changed := []byte("user note, not installer metadata\r\n")
			if mode == "changed-record" {
				foreign := record
				foreign.ShortcutHash = strings.Repeat("b", 64)
				changed, err = json.Marshal(foreign)
				if err != nil {
					t.Fatal(err)
				}
			}
			replaced := false
			err = finishDirectoryRemoval(record, func(entry string) error {
				if entry == path {
					if err := os.Remove(path); err != nil {
						return err
					}
					if mode == "directory" {
						if err := os.Mkdir(path, 0o700); err != nil {
							return err
						}
					} else if err := os.WriteFile(path, changed, 0o600); err != nil {
						return err
					}
					replaced = true
				}
				return removeInstallationEntry(record, entry)
			})
			if !replaced || err == nil {
				t.Fatalf("late ownership replacement was removed: reached=%v err=%v", replaced, err)
			}
			if mode == "directory" {
				info, err := os.Stat(path)
				if err != nil || !info.IsDir() {
					t.Fatalf("replacement directory disappeared: %v", err)
				}
			} else if data, err := os.ReadFile(path); err != nil || string(data) != string(changed) {
				t.Fatalf("replacement metadata disappeared: %v", err)
			}
			journal, err := readRemovalJournal(record.Directory, record.App.ID)
			if err != nil || !sameOwnership(journal, record) {
				t.Fatalf("original recovery ownership was lost: %v", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if _, err := uninstall(record.App.ID, env); err != nil {
				t.Fatalf("cannot resume after moving conflicting metadata: %v", err)
			}
		})
	}
}
