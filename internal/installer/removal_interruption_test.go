package installer

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRemovalProcessExitRecovery(t *testing.T) {
	for _, phase := range []string{"journal-ready", "record-removed", "root-removed"} {
		t.Run(phase, func(t *testing.T) {
			source, setup, env := fixture(t)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			draft := filepath.Join(filepath.Dir(env.root), "retained-user-draft.md")
			if err := os.WriteFile(draft, []byte("retained\r\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			self, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, self, "-test.run=^TestRemovalProcessExitHelper$")
			child.Env = append(os.Environ(), "VELOX_REMOVAL_CUT_DIRECTORY="+installed.Directory, "VELOX_REMOVAL_CUT="+phase)
			output, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 97 {
				t.Fatalf("child missed cut point: %v; %s", err, output)
			}
			if _, err := readRemovalJournal(installed.Directory, installed.AppID); err != nil {
				t.Fatalf("abrupt exit lost the recovery record: %v", err)
			}
			_, stateErr := os.Stat(filepath.Join(installed.Directory, stateFile))
			if phase == "journal-ready" && stateErr != nil {
				t.Fatalf("cut happened after record deletion: %v", stateErr)
			}
			if phase != "journal-ready" && !os.IsNotExist(stateErr) {
				t.Fatalf("cut did not delete the internal record: %v", stateErr)
			}
			if phase == "root-removed" {
				if _, err := os.Stat(installed.Directory); !os.IsNotExist(err) {
					t.Fatalf("root remained at post-removal cut: %v", err)
				}
			}
			if phase == "record-removed" {
				note := filepath.Join(installed.Directory, "late-user-file.md")
				if err := os.WriteFile(note, []byte("preserve me"), 0o600); err != nil {
					t.Fatal(err)
				}
				if _, err := uninstall(installed.AppID, env); err == nil {
					t.Fatal("recovery ignored a newly added user file")
				}
				if data, err := os.ReadFile(note); err != nil || string(data) != "preserve me" {
					t.Fatalf("recovery changed the user file: %v", err)
				}
				if err := os.Remove(note); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := uninstall(installed.AppID, env); err != nil {
				t.Fatalf("recovery after process exit failed: %v", err)
			}
			if _, err := os.Stat(removalJournalPath(installed.Directory, installed.AppID)); !os.IsNotExist(err) {
				t.Fatalf("completed recovery retained its journal: %v", err)
			}
			if data, err := os.ReadFile(draft); err != nil || string(data) != "retained\r\n" {
				t.Fatalf("outside draft changed: %v", err)
			}
			if _, err := install(source, setup, env); err != nil {
				t.Fatalf("new install remained blocked after recovery: %v", err)
			}
		})
	}
}

func TestRemovalProcessExitHelper(t *testing.T) {
	directory := os.Getenv("VELOX_REMOVAL_CUT_DIRECTORY")
	if directory == "" {
		t.Skip("isolated child-process fixture")
	}
	phase := os.Getenv("VELOX_REMOVAL_CUT")
	if phase != "journal-ready" && phase != "record-removed" && phase != "root-removed" {
		t.Fatal("invalid removal cut point")
	}
	record, err := readState(directory, "dev.velox.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range record.Files {
		if err := os.Remove(filepath.Join(directory, filepath.FromSlash(file.Path))); err != nil {
			t.Fatal(err)
		}
	}
	if err := finishDirectoryRemoval(record, func(path string) error {
		isState := path == filepath.Join(directory, stateFile)
		if isState && phase == "journal-ready" {
			os.Exit(97)
		}
		err := removeInstallationEntry(record, path)
		if err == nil && ((isState && phase == "record-removed") || (path == directory && phase == "root-removed")) {
			os.Exit(97)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t.Fatal("did not reach requested process exit")
}
