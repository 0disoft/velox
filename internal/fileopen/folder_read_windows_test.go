//go:build windows

package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestFolderTextLocalUTF8BoundsAndUnchangedBytes(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		text string
		err  error
	}{
		{"empty", nil, "", nil},
		{"unicode", []byte("hello \xed\x95\x9c\xea\xb8\x80"), "hello \xed\x95\x9c\xea\xb8\x80", nil},
		{"bom", []byte("\xef\xbb\xbfhello"), "hello", nil},
		{"exact", []byte(strings.Repeat("a", MaxTextBytes)), strings.Repeat("a", MaxTextBytes), nil},
		{"large", []byte(strings.Repeat("a", MaxTextBytes+1)), "", ErrTooLarge},
		{"invalid", []byte{0xff}, "", ErrUnsupported},
		{"binary", []byte{'a', 0}, "", ErrUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			name := "\xed\x95\x9c\xea\xb8\x80.txt"
			path := filepath.Join(root, name)
			if err := os.WriteFile(path, tc.data, 0o600); err != nil {
				t.Fatal(err)
			}
			snapshot, err := inspectSelectedFolder(root)
			if err != nil {
				t.Fatal(err)
			}
			result, err := readFolderText(root, snapshot.ID, name)
			if !errors.Is(err, tc.err) {
				t.Fatalf("got %v want %v", err, tc.err)
			}
			if err == nil && (result.Name != name || result.Text != tc.text || result.Bytes != len(tc.data) || result.Cancelled) {
				t.Fatal("result differs")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != string(tc.data) {
				t.Fatal("read changed file")
			}
		})
	}
}

func TestFolderTextRejectsTraversalAndUnsupportedChildren(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot, err := inspectSelectedFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"..", `..\outside.txt`, "../outside.txt", `child\file.txt`, `C:\outside.txt`, "file.txt:stream", "NUL", "file.", "child"} {
		if _, err := readFolderText(root, snapshot.ID, name); !errors.Is(err, ErrUnsupported) {
			t.Fatal(name, err)
		}
	}
	if _, err := readFolderText(root, snapshot.ID, "missing.txt"); err == nil {
		t.Fatal("missing file accepted")
	}
	path := filepath.Join(root, "offline.txt")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Fatal(err)
	}
	defer windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL)
	if _, err := readFolderText(root, snapshot.ID, "offline.txt"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("offline file read", err)
	}
}

func TestFolderTextRejectsHardLinksAndSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(outside, []byte("outside private text"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := inspectSelectedFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Link(outside, filepath.Join(root, "hard.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := readFolderText(root, snapshot.ID, "hard.txt"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("hard link read", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err != nil {
		t.Skip("symlink creation unavailable")
	}
	if _, err := readFolderText(root, snapshot.ID, "link.txt"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("symlink read", err)
	}
}

func TestFolderTextRejectsReplacedDirectoryAndUsesPinnedHandle(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "selected")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("selected text"), 0o600); err != nil {
		t.Fatal(err)
	}
	handle, snapshot, err := openSelectedFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	if err := os.Rename(root, filepath.Join(base, "original")); err != nil {
		t.Fatal(err)
	}
	if _, err := readFolderText(root, snapshot.ID, "notes.txt"); !errors.Is(err, ErrFolderTarget) {
		t.Fatal("deleted path accepted", err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("replacement text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readFolderText(root, snapshot.ID, "notes.txt"); !errors.Is(err, ErrFolderTarget) {
		t.Fatal("replacement accepted", err)
	}
	result, err := readFolderTextAt(handle, "notes.txt")
	if err != nil || result.Text != "selected text" {
		t.Fatal("read followed replaced path", result, err)
	}
}
