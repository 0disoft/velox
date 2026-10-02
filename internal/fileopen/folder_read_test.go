package fileopen

import (
	"errors"
	"testing"
)

func TestFolderTextReadIsDeferredBoundToIdentityAndSerialized(t *testing.T) {
	f, queue, _ := folderFixture()
	selected := selectFixture(t, f, queue)
	reads := 0
	f.ReadText = func(path string, id DirectoryID, name string) (Result, error) {
		reads++
		if path != "private-folder" || id != (DirectoryID{1, 2, 3}) || name != "notes.txt" {
			t.Fatal("read parameters differ")
		}
		return Result{Name: name, Text: "hello", Bytes: 5}, nil
	}
	var result Result
	if err := f.OpenText(selected.Target, "notes.txt", func(r Result, err error) {
		if err != nil {
			t.Fatal(err)
		}
		result = r
	}); err != nil {
		t.Fatal(err)
	}
	if reads != 0 {
		t.Fatal("read ran inside callback")
	}
	if err := f.Select(func(FolderResult, error) {}); err != ErrBusy {
		t.Fatal("selection overlapped read", err)
	}
	f.Read = func(string, DirectoryID) (FolderListing, error) {
		t.Fatal("list overlapped read")
		return FolderListing{}, nil
	}
	if err := f.List(selected.Target, func(FolderListing, error) {}); err != ErrBusy {
		t.Fatal("list overlapped read", err)
	}
	(*queue)[1]()
	if reads != 1 || result.Text != "hello" || result.Name != "notes.txt" || result.Bytes != 5 {
		t.Fatal(result)
	}
}

func TestFolderTextRejectsNamesBeforeIO(t *testing.T) {
	f, queue, _ := folderFixture()
	selected := selectFixture(t, f, queue)
	f.ReadText = func(string, DirectoryID, string) (Result, error) {
		t.Fatal("invalid read reached disk")
		return Result{}, nil
	}
	for _, name := range []string{"", ".", "..", "../outside.txt", `child\file.txt`, `C:\file.txt`, "file.txt:stream", "NUL", "CON.txt", "file.", "file ", "bad\x00.txt"} {
		if err := f.OpenText(selected.Target, name, func(Result, error) {}); !errors.Is(err, ErrUnsupported) {
			t.Fatal(name, err)
		}
	}
	if len(*queue) != 1 {
		t.Fatal("invalid name queued IO")
	}
}

func TestFolderTextSuppressesRevokedOrInactiveResults(t *testing.T) {
	for _, mode := range []string{"wrong-token", "released-before", "released-during", "cleared-during", "generation-before", "generation-during", "inactive-during", "replaced-during", "read-failure"} {
		t.Run(mode, func(t *testing.T) {
			f, queue, generation := folderFixture()
			selected := selectFixture(t, f, queue)
			active, reads := true, 0
			f.Active = func() bool { return active }
			f.ReadText = func(string, DirectoryID, string) (Result, error) {
				reads++
				switch mode {
				case "released-during":
					f.ReleaseTarget(selected.Target)
				case "cleared-during":
					f.ClearTarget()
				case "generation-during":
					*generation++
				case "inactive-during":
					active = false
				case "replaced-during":
					f.mu.Lock()
					f.target.id++
					f.mu.Unlock()
				case "read-failure":
					return Result{Text: "private"}, ErrFolderTarget
				}
				return Result{Name: "notes.txt", Text: "private"}, nil
			}
			target := selected.Target
			if mode == "wrong-token" {
				target++
			}
			var got error
			if err := f.OpenText(target, "notes.txt", func(r Result, err error) {
				got = err
				if r.Text != "" || r.Name != "" {
					t.Fatal("stale content leaked")
				}
			}); err != nil {
				t.Fatal(err)
			}
			if mode == "released-before" {
				f.ReleaseTarget(selected.Target)
			}
			if mode == "generation-before" {
				*generation++
			}
			(*queue)[1]()
			if !errors.Is(got, ErrFolderTarget) && !errors.Is(got, ErrInactive) {
				t.Fatal("stale read succeeded", got)
			}
			if (mode == "wrong-token" || mode == "released-before" || mode == "generation-before") && reads != 0 {
				t.Fatal("stale read started")
			}
		})
	}
}
