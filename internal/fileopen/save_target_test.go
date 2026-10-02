package fileopen

import (
	"crypto/sha256"
	"errors"
	"testing"
)

func TestSaveTargetLifecycle(t *testing.T) {
	var queued func()
	generation := uint64(1)
	selected, writes := 0, 0
	path, content := "private-selection", ""
	s := &Saver{Dispatch: func(fn func()) { queued = fn }, Active: func() bool { return true }, Generation: func() uint64 { return generation },
		Select: func(string) (string, error) { selected++; return path, nil },
		Write: func(p, text string) (SaveResult, error) {
			content = text
			return SaveResult{Name: "notes.txt", Bytes: len(text)}, nil
		},
		Snapshot: func(string) (FileVersion, error) {
			return FileVersion{Size: uint64(len(content)), Digest: sha256.Sum256([]byte(content))}, nil
		},
		WriteVersion: func(p, text string, version FileVersion) (SaveResult, error) {
			writes++
			if p != "private-selection" || version.Digest != sha256.Sum256([]byte(content)) {
				t.Fatal("wrong target or stale baseline")
			}
			content = text
			return SaveResult{Name: "notes.txt", Bytes: len(text)}, nil
		}}
	var target uint32
	if err := s.SaveAs("first", "notes.txt", func(r SaveResult, e error) {
		if e != nil || r.Target == 0 {
			t.Fatal(r, e)
		}
		target = r.Target
	}); err != nil {
		t.Fatal(err)
	}
	queued()
	for _, text := range []string{"second", "third"} {
		if err := s.SaveTo(text, target, func(r SaveResult, e error) {
			if e != nil || r.Target != target {
				t.Fatal(r, e)
			}
		}); err != nil {
			t.Fatal(err)
		}
		queued()
	}
	if selected != 1 || writes != 2 {
		t.Fatal("Save reopened dialog or skipped writes")
	}
	path = ""
	if err := s.SaveAs("cancelled", "notes.txt", func(r SaveResult, e error) {
		if e != nil || !r.Cancelled {
			t.Fatal(r, e)
		}
	}); err != nil {
		t.Fatal(err)
	}
	queued()
	if s.target == nil || s.target.id != target {
		t.Fatal("cancel discarded old target")
	}
	if err := s.SaveTo("revoked", target, func(r SaveResult, e error) {
		if !errors.Is(e, ErrTarget) {
			t.Fatal(r, e)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReleaseTarget(target); err != nil {
		t.Fatal(err)
	}
	queued()
	if writes != 2 {
		t.Fatal("revoked queued save wrote a file")
	}
	path = "private-selection"
	if err := s.SaveAs("fresh", "notes.txt", func(r SaveResult, e error) {
		if e != nil || r.Target == target {
			t.Fatal(r, e)
		}
		target = r.Target
	}); err != nil {
		t.Fatal(err)
	}
	queued()
	generation++
	if err := s.SaveTo("stale", target, func(r SaveResult, e error) {
		if !errors.Is(e, ErrTarget) {
			t.Fatal(r, e)
		}
	}); err != nil {
		t.Fatal(err)
	}
	queued()
	if writes != 2 {
		t.Fatal("previous document wrote a file")
	}
	s.ClearTarget()
	if s.target != nil {
		t.Fatal("target not cleared")
	}
}

func TestSaveAsDoesNotConnectChangedReadback(t *testing.T) {
	var queued func()
	s := &Saver{Dispatch: func(fn func()) { queued = fn }, Active: func() bool { return true }, Generation: func() uint64 { return 1 },
		Select: func(string) (string, error) { return "private-selection", nil },
		Write:  func(string, string) (SaveResult, error) { return SaveResult{Name: "notes.txt"}, nil },
		Snapshot: func(string) (FileVersion, error) {
			return FileVersion{Size: 5, Digest: sha256.Sum256([]byte("other"))}, nil
		},
		WriteVersion: func(string, string, FileVersion) (SaveResult, error) {
			t.Fatal("unexpected reuse")
			return SaveResult{}, nil
		},
	}
	if err := s.SaveAs("first", "notes.txt", func(r SaveResult, e error) {
		if !errors.Is(e, ErrConflict) {
			t.Fatal(r, e)
		}
	}); err != nil {
		t.Fatal(err)
	}
	queued()
	if s.target != nil {
		t.Fatal("changed output was connected")
	}
}
