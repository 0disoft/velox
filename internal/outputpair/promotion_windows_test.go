package outputpair

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func lockAgainstRename(t *testing.T, path string) func() {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	unlock := func() {
		if !closed {
			closed = true
			if err := syscall.CloseHandle(handle); err != nil {
				t.Error(err)
			}
		}
	}
	t.Cleanup(unlock)
	return unlock
}

func TestRecoverRetriesLockedArchiveAfterDirectoryRestoration(t *testing.T) {
	root := t.TempDir()
	directory, archive := filepath.Join(root, "app"), filepath.Join(root, "app.zip")
	write(t, filepath.Join(directory+".previous", "old.txt"))
	write(t, archive+".previous")
	unlock := lockAgainstRename(t, archive+".previous")
	if err := Recover(directory, archive); err == nil {
		t.Fatal("restored locked archive")
	}
	assertExists(t, filepath.Join(directory, "old.txt"))
	assertExists(t, archive+".previous")
	assertMissing(t, archive)
	unlock()
	if err := Recover(directory, archive); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(directory, "old.txt"))
	assertExists(t, archive)
	assertMissing(t, archive+".previous")
}

func TestPromotePreservesPairWhenArchiveIsLocked(t *testing.T) {
	root := t.TempDir()
	directory, archive := filepath.Join(root, "app"), filepath.Join(root, "app.zip")
	stageDirectory, stageArchive := filepath.Join(root, "stage"), filepath.Join(root, "stage.zip")
	write(t, filepath.Join(directory, "old.txt"))
	write(t, archive)
	write(t, filepath.Join(stageDirectory, "new.txt"))
	if err := os.WriteFile(stageArchive, []byte("new archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	unlock := lockAgainstRename(t, archive)
	if err := Promote(directory, archive, stageDirectory, stageArchive); err == nil {
		t.Fatal("promoted over locked archive")
	}
	assertExists(t, filepath.Join(directory, "old.txt"))
	data, err := os.ReadFile(archive)
	if err != nil || string(data) != "fixture" {
		t.Fatalf("previous archive changed: %q, %v", data, err)
	}
	unlock()
	if err := Promote(directory, archive, stageDirectory, stageArchive); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(directory, "new.txt"))
	assertMissing(t, directory+".previous")
	assertMissing(t, archive+".previous")
}
