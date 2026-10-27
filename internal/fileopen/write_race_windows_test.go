//go:build windows

package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestSaveRetainsFileReplacedAfterFinalIdentityCheck(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	foreign := filepath.Join(root, "external.txt")
	for name, text := range map[string]string{path: "original", foreign: "external"} {
		if err := os.WriteFile(name, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(foreign, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotSelected(path)
	if err != nil {
		t.Fatal(err)
	}
	other, err := snapshotSelected(foreign)
	if err != nil || before.Size != other.Size || before.Modified != other.Modified {
		t.Fatalf("fixture must match size and timestamp: before=%+v other=%+v err=%v", before, other, err)
	}
	var backupPath string
	_, err = writeText(path, "our new text", nil, func(path, replacement, backup string) bool {
		backupPath = backup
		if err := os.Rename(path, filepath.Join(root, "original-moved.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(foreign, path); err != nil {
			t.Fatal(err)
		}
		return replaceSavedFile(path, replacement, backup)
	})
	if !errors.Is(err, ErrRecovery) {
		t.Errorf("raced replacement returned %v, want recovery required", err)
	}
	data, readErr := os.ReadFile(backupPath)
	if readErr != nil || string(data) != "external" {
		t.Fatalf("external changes were lost: backup=%q, err=%v", data, readErr)
	}
	data, readErr = os.ReadFile(path)
	if readErr != nil || string(data) != "our new text" {
		t.Fatalf("new text was not retained: %q, %v", data, readErr)
	}
}

func TestBackupComparisonUsesPostRenameIdentityAndPreSaveMetadata(t *testing.T) {
	before := windows.ByHandleFileInformation{VolumeSerialNumber: 1, FileIndexLow: 10, FileSizeLow: 8, NumberOfLinks: 1}
	current := before
	current.FileIndexLow = 20 // A FAT rename can change the ID of the same open file.
	if !matchesSaveBackup(current, current, before) {
		t.Fatal("rejected matching live handles after rename changed the original ID")
	}
	foreign := current
	foreign.FileIndexLow++
	if matchesSaveBackup(foreign, current, before) {
		t.Fatal("accepted a different file with matching size and timestamp")
	}
	changed := current
	changed.FileSizeLow++
	if matchesSaveBackup(changed, changed, before) {
		t.Fatal("accepted a size change to the same file")
	}
	changed = current
	changed.LastWriteTime.LowDateTime++
	if matchesSaveBackup(changed, changed, before) {
		t.Fatal("accepted a timestamp change to the same file")
	}
}
