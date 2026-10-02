package fileopen

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func folderFixture() (*Folder, *[]func(), *uint64) {
	queue := []func(){}
	generation := uint64(1)
	f := &Folder{Dispatch: func(fn func()) { queue = append(queue, fn) }, Active: func() bool { return true },
		Generation: func() uint64 { return generation }, Choose: func() (string, error) { return "private-folder", nil },
		Inspect: func(string) (FolderSnapshot, error) {
			return FolderSnapshot{Name: "Selected", ID: DirectoryID{1, 2, 3}}, nil
		}}
	return f, &queue, &generation
}

func selectFixture(t *testing.T, f *Folder, queue *[]func()) FolderResult {
	t.Helper()
	var result FolderResult
	if err := f.Select(func(r FolderResult, err error) {
		if err != nil {
			t.Fatal(err)
		}
		result = r
	}); err != nil {
		t.Fatal(err)
	}
	(*queue)[len(*queue)-1]()
	return result
}

func TestFolderSelectionCancellationReplacementAndPrivacy(t *testing.T) {
	f, queue, _ := folderFixture()
	first := selectFixture(t, f, queue)
	if first.Target == 0 || first.Name != "Selected" {
		t.Fatal(first)
	}
	body, _ := json.Marshal(first)
	if strings.Contains(string(body), "private-folder") || strings.Contains(string(body), "path") {
		t.Fatal("private path leaked")
	}
	f.Choose = func() (string, error) { return "", nil }
	cancelled := selectFixture(t, f, queue)
	if !cancelled.Cancelled || cancelled.Target != 0 || f.target.id != first.Target {
		t.Fatal("cancel revoked target")
	}
	f.Choose = func() (string, error) { return "other-folder", nil }
	second := selectFixture(t, f, queue)
	if second.Target <= first.Target || f.ReleaseTarget(first.Target) != ErrFolderTarget {
		t.Fatal("token reused")
	}
	if err := f.ReleaseTarget(second.Target); err != nil {
		t.Fatal(err)
	}
	if f.target != nil || f.ReleaseTarget(second.Target) != ErrFolderTarget {
		t.Fatal("release failed")
	}
}

func TestFolderSelectionGenerationAndClearInvalidateQueuedWork(t *testing.T) {
	for _, mode := range []string{"before-dispatch", "during-dialog", "during-inspect", "clear"} {
		t.Run(mode, func(t *testing.T) {
			f, queue, generation := folderFixture()
			var got error
			chosen := false
			f.Choose = func() (string, error) {
				chosen = true
				if mode == "during-dialog" {
					*generation++
				}
				return "folder", nil
			}
			f.Inspect = func(string) (FolderSnapshot, error) {
				if mode == "during-inspect" {
					*generation++
				}
				return FolderSnapshot{}, nil
			}
			if err := f.Select(func(_ FolderResult, err error) { got = err }); err != nil {
				t.Fatal(err)
			}
			if mode == "before-dispatch" {
				*generation++
			}
			if mode == "clear" {
				f.ClearTarget()
			}
			(*queue)[0]()
			if got == nil || f.target != nil {
				t.Fatal("stale work connected", got)
			}
			if mode == "before-dispatch" && chosen {
				t.Fatal("stale dialog opened")
			}
		})
	}
}

func TestFolderBusyFailureAndTokenExhaustion(t *testing.T) {
	f, queue, _ := folderFixture()
	first := selectFixture(t, f, queue)
	f.Inspect = func(string) (FolderSnapshot, error) { return FolderSnapshot{}, ErrFolderUnsupported }
	var got error
	if err := f.Select(func(_ FolderResult, err error) { got = err }); err != nil {
		t.Fatal(err)
	}
	if err := f.Select(func(FolderResult, error) {}); !errors.Is(err, ErrBusy) {
		t.Fatal("duplicate selection queued")
	}
	(*queue)[1]()
	if got != ErrFolderUnsupported || f.target.id != first.Target {
		t.Fatal("failure lost old target")
	}
	f.Inspect = func(string) (FolderSnapshot, error) { return FolderSnapshot{}, nil }
	f.serial = ^uint32(0)
	if err := f.Select(func(_ FolderResult, err error) { got = err }); err != nil {
		t.Fatal(err)
	}
	(*queue)[2]()
	if got != ErrBusy || f.target.id != first.Target {
		t.Fatal("exhausted serial wrapped")
	}
}

func TestFolderReleaseCannotCrossDocumentGeneration(t *testing.T) {
	f, queue, generation := folderFixture()
	selected := selectFixture(t, f, queue)
	*generation++
	if f.ReleaseTarget(selected.Target) != ErrFolderTarget {
		t.Fatal("stale token accepted")
	}
	f.ClearTarget()
	if f.target != nil {
		t.Fatal("target retained")
	}
}
