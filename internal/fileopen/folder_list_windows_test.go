//go:build windows

package fileopen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func listFixture(t *testing.T, path string) FolderListing {
	t.Helper()
	snapshot, err := inspectSelectedFolder(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := listSelectedFolder(path, snapshot.ID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > MaxFolderResultBytes || len(result.Entries) > MaxFolderEntries {
		t.Fatal("bounds exceeded", len(encoded), err)
	}
	return result
}

func TestFolderListEmptyUnicodeAndImmediateEntriesOnly(t *testing.T) {
	root := t.TempDir()
	empty := listFixture(t, root)
	if empty.Entries == nil || len(empty.Entries) != 0 || empty.Truncated {
		t.Fatal(empty)
	}
	name := "\xed\x95\x9c\xea\xb8\x80.txt"
	if err := os.WriteFile(filepath.Join(root, name), []byte("private contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "child", "nested.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := listFixture(t, root)
	kinds := map[string]string{}
	for _, item := range result.Entries {
		kinds[item.Name] = item.Kind
	}
	if len(kinds) != 2 || kinds[name] != "file" || kinds["child"] != "directory" || result.Truncated {
		t.Fatal(result)
	}
	body, _ := json.Marshal(result)
	if strings.Contains(string(body), "private contents") || strings.Contains(string(body), "nested.txt") || strings.Contains(string(body), "path") {
		t.Fatal("contents or path leaked")
	}
}

func TestFolderListEntryAndEscapedByteLimits(t *testing.T) {
	for _, mode := range []string{"entries", "bytes"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			count := MaxFolderEntries + 10
			if mode == "bytes" {
				count = 40
			}
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("%03d.txt", i)
				if mode == "bytes" {
					name = fmt.Sprintf("%03d", i) + strings.Repeat("&", 180) + ".txt"
				}
				if err := os.WriteFile(filepath.Join(root, name), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			result := listFixture(t, root)
			if !result.Truncated || len(result.Entries) == 0 || len(result.Entries) >= count {
				t.Fatal("not bounded", result)
			}
			if mode == "entries" && len(result.Entries) != MaxFolderEntries {
				t.Fatal("entry limit differs")
			}
		})
	}
}

func TestFolderListRejectsDeletedReplacedOrLinkedTargets(t *testing.T) {
	for _, mode := range []string{"deleted", "replaced", "linked"} {
		t.Run(mode, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "selected")
			if err := os.Mkdir(root, 0o700); err != nil {
				t.Fatal(err)
			}
			snapshot, err := inspectSelectedFolder(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(root, filepath.Join(base, "original")); err != nil {
				t.Fatal(err)
			}
			if mode == "replaced" {
				if err := os.Mkdir(root, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "linked" {
				if err := os.Symlink(filepath.Join(base, "original"), root); err != nil {
					t.Skip("symlink unavailable")
				}
			}
			if _, err := listSelectedFolder(root, snapshot.ID); !errors.Is(err, ErrFolderTarget) {
				t.Fatal("stale folder listed", err)
			}
		})
	}
}

func TestFolderListSkipsOfflineAndReparseEntriesWithoutFollowing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "offline.txt")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Fatal(err)
	}
	defer windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL)
	result := listFixture(t, root)
	if len(result.Entries) != 0 || result.Skipped != 1 {
		t.Fatal("offline entry exposed", result)
	}
	link := filepath.Join(root, "linked-folder")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skip("symlink unavailable")
	}
	result = listFixture(t, root)
	if len(result.Entries) != 0 || result.Skipped != 2 {
		t.Fatal("link followed", result)
	}
}
