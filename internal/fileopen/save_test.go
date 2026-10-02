package fileopen

import (
	"errors"
	"strings"
	"testing"
)

func TestSaveValidation(t *testing.T) {
	for _, name := range []string{"", "../a.txt", `C:\a.txt`, "x:stream", "NUL.txt", "COM1", "name.", "name ", "a\x00b", strings.Repeat("a", 241)} {
		if ValidateSaveName(name) == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	for _, text := range []string{"a\x00b", string([]byte{0xff}), strings.Repeat("x", MaxTextBytes+1)} {
		if ValidateSaveText(text) == nil {
			t.Fatal("accepted invalid text")
		}
	}
	if ValidateSaveName("notes.md") != nil || ValidateSaveText(strings.Repeat("x", MaxTextBytes)) != nil {
		t.Fatal("valid boundary rejected")
	}
}

func TestSaveDeferredCancellationAndNavigation(t *testing.T) {
	for _, mode := range []string{"save", "cancel", "before", "during", "closed", "write-error", "select-error"} {
		t.Run(mode, func(t *testing.T) {
			var queued func()
			generation, active := uint64(1), true
			writes, replies := 0, 0
			s := &Saver{Dispatch: func(fn func()) { queued = fn }, Active: func() bool { return active }, Generation: func() uint64 { return generation },
				Select: func(name string) (string, error) {
					if name != "notes.md" {
						t.Fatal(name)
					}
					if mode == "during" {
						generation++
					}
					if mode == "cancel" {
						return "", nil
					}
					if mode == "select-error" {
						return "", ErrUnsupported
					}
					return "selected", nil
				}, Write: func(path, text string) (SaveResult, error) {
					writes++
					if path != "selected" || text != "hello" {
						t.Fatal("wrong write")
					}
					if mode == "write-error" {
						return SaveResult{}, ErrRecovery
					}
					return SaveResult{Name: "notes.md", Bytes: 5}, nil
				}}
			if err := s.Save("hello", "notes.md", func(r SaveResult, e error) {
				replies++
				switch mode {
				case "save":
					if e != nil || r.Bytes != 5 {
						t.Fatal(r, e)
					}
				case "cancel":
					if e != nil || !r.Cancelled {
						t.Fatal(r, e)
					}
				case "write-error":
					if !errors.Is(e, ErrRecovery) {
						t.Fatal(e)
					}
				case "select-error":
					if !errors.Is(e, ErrUnsupported) {
						t.Fatal(e)
					}
				default:
					if !errors.Is(e, ErrInactive) {
						t.Fatal(e)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			if writes != 0 {
				t.Fatal("write during admission")
			}
			if err := s.Save("hello", "notes.md", func(SaveResult, error) {}); !errors.Is(err, ErrBusy) {
				t.Fatal(err)
			}
			if mode == "before" {
				generation++
			}
			if mode == "closed" {
				active = false
			}
			queued()
			wantWrites := 0
			if mode == "save" || mode == "write-error" {
				wantWrites = 1
			}
			if replies != 1 || writes != wantWrites || s.pending.Load() {
				t.Fatal("incorrect completion or retained busy state")
			}
		})
	}
}
