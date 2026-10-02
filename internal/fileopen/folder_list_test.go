package fileopen

import (
	"errors"
	"testing"
)

func TestFolderListUsesOnlyConnectedIdentityAndDeferredQueue(t *testing.T) {
	f, queue, _ := folderFixture()
	selected := selectFixture(t, f, queue)
	reads := 0
	f.Read = func(path string, id DirectoryID) (FolderListing, error) {
		reads++
		if path != "private-folder" || id != (DirectoryID{1, 2, 3}) {
			t.Fatal("identity changed")
		}
		return FolderListing{Entries: []FolderEntry{{"a.txt", "file"}}}, nil
	}
	var result FolderListing
	if err := f.List(selected.Target, func(r FolderListing, err error) {
		if err != nil {
			t.Fatal(err)
		}
		result = r
	}); err != nil {
		t.Fatal(err)
	}
	if reads != 0 {
		t.Fatal("read ran in caller callback")
	}
	if err := f.Select(func(FolderResult, error) {}); err != ErrBusy {
		t.Fatal("selection overlapped list")
	}
	(*queue)[1]()
	if reads != 1 || len(result.Entries) != 1 {
		t.Fatal(result)
	}
}

func TestFolderListRejectsStaleRevokedAndReplacedTargets(t *testing.T) {
	for _, mode := range []string{"wrong-token", "release-before-dispatch", "release-during-read", "generation", "clear-during-read", "identity-error"} {
		t.Run(mode, func(t *testing.T) {
			f, queue, generation := folderFixture()
			selected := selectFixture(t, f, queue)
			reads := 0
			f.Read = func(string, DirectoryID) (FolderListing, error) {
				reads++
				if mode == "release-during-read" {
					f.ReleaseTarget(selected.Target)
				}
				if mode == "clear-during-read" {
					f.ClearTarget()
				}
				if mode == "identity-error" {
					return FolderListing{}, ErrFolderTarget
				}
				return FolderListing{Entries: []FolderEntry{{"private", "file"}}}, nil
			}
			target := selected.Target
			if mode == "wrong-token" {
				target++
			}
			var got error
			if err := f.List(target, func(r FolderListing, err error) {
				got = err
				if len(r.Entries) != 0 {
					t.Fatal("stale result leaked")
				}
			}); err != nil {
				t.Fatal(err)
			}
			if mode == "release-before-dispatch" {
				f.ReleaseTarget(selected.Target)
			}
			if mode == "generation" {
				*generation++
			}
			(*queue)[1]()
			if !errors.Is(got, ErrFolderTarget) && !errors.Is(got, ErrInactive) {
				t.Fatal("stale target accepted", got)
			}
			if (mode == "wrong-token" || mode == "release-before-dispatch" || mode == "generation") && reads != 0 {
				t.Fatal("stale read started")
			}
		})
	}
}
