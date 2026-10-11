package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func removalFixture(t *testing.T, directory string) (string, File) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "owned.txt")
	data := []byte("owned bytes\r\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	return path, File{Path: "owned.txt", Bytes: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}
}

func TestWindowsRemovalPinsVerifiedFileAndParents(t *testing.T) {
	for _, longPath := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "long-path"}[longPath], func(t *testing.T) {
			parent := filepath.Join(t.TempDir(), "installation")
			if longPath {
				for len(parent) < 280 {
					parent = filepath.Join(parent, strings.Repeat("segment", 8))
				}
			}
			path, owned := removalFixture(t, parent)
			locked, err := openRemovalFile(path, owned)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = locked.close() })
			for _, operation := range []func() error{
				func() error { return os.WriteFile(path, []byte("user changes"), 0o600) },
				func() error { return os.Rename(path, path+".moved") },
				func() error { return os.Remove(path) },
				func() error { return os.Rename(parent, parent+".moved") },
			} {
				if err := operation(); err == nil {
					t.Fatal("verified file or its parent could be changed before handle deletion")
				}
			}
			// A concurrent user file is not owned or deleted by the handle operation.
			note := filepath.Join(parent, "user-note.txt")
			if err := os.WriteFile(note, []byte("keep me"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := errors.Join(locked.remove(), locked.close()); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("verified file was not deleted: %v", err)
			}
			if data, err := os.ReadFile(note); err != nil || string(data) != "keep me" {
				t.Fatalf("concurrent note was not preserved: %v", err)
			}
			if err := os.Rename(parent, parent+".moved"); err != nil {
				t.Fatalf("removal leaked a parent lock: %v", err)
			}
		})
	}
}

func TestWindowsRemovalRefusesOpenWriter(t *testing.T) {
	path, owned := removalFixture(t, t.TempDir())
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := removeRecordedFile(path, owned); !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		windows.CloseHandle(handle)
		t.Fatalf("removal ignored a live writer: %v", err)
	}
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	if err := removeRecordedFile(path, owned); err != nil {
		t.Fatalf("cannot retry after writer closes: %v", err)
	}
}

func TestWindowsRemovalPreservesLinkedPath(t *testing.T) {
	for _, parentLink := range []bool{false, true} {
		t.Run(map[bool]string{false: "file-link", true: "directory-link"}[parentLink], func(t *testing.T) {
			root := t.TempDir()
			outside, owned := removalFixture(t, filepath.Join(root, "outside"))
			path := filepath.Join(root, "link.txt")
			linkTarget := outside
			if parentLink {
				path = filepath.Join(root, "link")
				linkTarget = filepath.Dir(outside)
			}
			if err := os.Symlink(linkTarget, path); err != nil {
				t.Skipf("native symlink creation unavailable: %v", err)
			}
			link := path
			if parentLink {
				path = filepath.Join(path, filepath.Base(outside))
			}
			if err := removeRecordedFile(path, owned); err == nil {
				t.Fatal("removal followed a linked path")
			}
			if _, err := os.Lstat(link); err != nil {
				t.Fatalf("removal deleted the link: %v", err)
			}
			if data, err := os.ReadFile(outside); err != nil || string(data) != "owned bytes\r\n" {
				t.Fatalf("linked outside file was changed: %v", err)
			}
		})
	}
}
