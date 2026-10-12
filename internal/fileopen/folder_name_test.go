package fileopen

import (
	"errors"
	"strings"
	"testing"
)

func TestFolderTextReadsLongWindowsBasenamesWithoutWideningSaves(t *testing.T) {
	for _, name := range []string{strings.Repeat("a", 251) + ".txt", strings.Repeat("\uD55C", 251) + ".txt", strings.Repeat("\U0001F642", 125) + "a.txt"} {
		f, queue, _ := folderFixture()
		selected := selectFixture(t, f, queue)
		f.ReadText = func(_ string, _ DirectoryID, got string) (Result, error) {
			if got != name {
				t.Fatal("basename changed")
			}
			return Result{Name: got, Text: "text", Bytes: 4}, nil
		}
		var result Result
		if err := f.OpenText(selected.Target, name, func(r Result, err error) {
			if err != nil {
				t.Fatal(err)
			}
			result = r
		}); err != nil {
			t.Fatalf("valid long basename rejected: bytes=%d: %v", len(name), err)
		}
		(*queue)[1]()
		if result.Name != name || result.Text != "text" {
			t.Fatal(result)
		}
		if !errors.Is(ValidateSaveName(name), ErrUnsupported) {
			t.Fatal("save bound widened")
		}
	}
}

func TestFolderTextRejectsOverlongUTF16BasenamesBeforeIO(t *testing.T) {
	f, queue, _ := folderFixture()
	selected := selectFixture(t, f, queue)
	f.ReadText = func(string, DirectoryID, string) (Result, error) { t.Fatal("invalid name read"); return Result{}, nil }
	for _, name := range []string{strings.Repeat("a", 256), strings.Repeat("\U0001F642", 128), strings.Repeat("\uD55C", 256), strings.Repeat("a", 1021), "bad\xff.txt"} {
		if err := f.OpenText(selected.Target, name, func(Result, error) {}); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("invalid basename accepted: bytes=%d %v", len(name), err)
		}
	}
	if len(*queue) != 1 {
		t.Fatal("invalid name queued IO")
	}
}
