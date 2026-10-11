package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsFolderFailureReselectionAndReleaseFlow(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "selected")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("original text\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, queue, _ := folderFixture()
	f.Choose = func() (string, error) { return root, nil }
	f.Inspect, f.Read, f.ReadText = inspectSelectedFolder, listSelectedFolder, readFolderText
	read := func(target uint32) (Result, error) {
		t.Helper()
		var result Result
		var failure error
		called := false
		if err := f.OpenText(target, "notes.txt", func(value Result, err error) {
			called, result, failure = true, value, err
		}); err != nil {
			return Result{}, err
		}
		(*queue)[len(*queue)-1]()
		if !called {
			t.Fatal("read callback did not run")
		}
		return result, failure
	}
	list := func(target uint32) (FolderListing, error) {
		t.Helper()
		var result FolderListing
		var failure error
		called := false
		if err := f.List(target, func(value FolderListing, err error) {
			called, result, failure = true, value, err
		}); err != nil {
			return FolderListing{}, err
		}
		(*queue)[len(*queue)-1]()
		if !called {
			t.Fatal("list callback did not run")
		}
		return result, failure
	}
	first := selectFixture(t, f, queue)
	if result, err := read(first.Target); err != nil || result.Text != "original text\r\n" {
		t.Fatalf("initial read failed: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := read(first.Target); err == nil {
		t.Fatal("deleted file read succeeded")
	}
	if result, err := list(first.Target); err != nil || len(result.Entries) != 0 {
		t.Fatalf("a file read failure prevented refresh: %v", err)
	}
	if err := os.WriteFile(path, []byte("restored text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result, err := read(first.Target); err != nil || result.Text != "restored text" {
		t.Fatalf("cannot retry after a file reappears: %v", err)
	}
	if err := os.Rename(root, filepath.Join(base, "original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement \xed\x95\x9c\xea\xb8\x80\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result, err := list(first.Target); !errors.Is(err, ErrFolderTarget) || len(result.Entries) != 0 {
		t.Fatalf("stale token listed a replacement: %v", err)
	}
	if result, err := read(first.Target); !errors.Is(err, ErrFolderTarget) || result.Text != "" {
		t.Fatalf("stale token read replacement contents: %v", err)
	}
	second := selectFixture(t, f, queue)
	if second.Target <= first.Target {
		t.Fatal("reselection reused the stale token")
	}
	if result, err := list(second.Target); err != nil || len(result.Entries) != 1 {
		t.Fatalf("reselection did not recover the list: %v", err)
	}
	if result, err := read(second.Target); err != nil || result.Text != "replacement \xed\x95\x9c\xea\xb8\x80\r\n" {
		t.Fatalf("reselection did not recover reading: %v", err)
	}
	f.Choose = func() (string, error) { return "", nil }
	if cancelled := selectFixture(t, f, queue); !cancelled.Cancelled {
		t.Fatal("injected picker cancellation was not reported")
	}
	if result, err := read(second.Target); err != nil || result.Text == "" {
		t.Fatalf("cancellation lost the existing selection: %v", err)
	}
	if err := f.ReleaseTarget(second.Target); err != nil {
		t.Fatal(err)
	}
	if _, err := read(second.Target); !errors.Is(err, ErrFolderTarget) {
		t.Fatalf("released folder stayed accessible: %v", err)
	}
	f.Choose = func() (string, error) { return root, nil }
	third := selectFixture(t, f, queue)
	if third.Target <= second.Target {
		t.Fatal("selection after release reused its token")
	}
	if _, err := read(third.Target); err != nil {
		t.Fatalf("cannot reselect after release: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("completed read leaked its file deletion lock: %v", err)
	}
}
