package fileopen

import (
	"errors"
	"testing"
)

func TestSelectionIsDeferredAndBusyUntilCompletion(t *testing.T) {
	var queued func()
	selected, read := 0, 0
	opener := &Opener{Dispatch: func(fn func()) { queued = fn }, Active: func() bool { return true }, Generation: func() uint64 { return 1 },
		Select: func() (string, error) { selected++; return "chosen", nil }, Read: func(path string) (Result, error) {
			read++
			if path != "chosen" {
				t.Fatal("wrong selection")
			}
			return Result{Text: "hello"}, nil
		}}
	results := 0
	if err := opener.Open(func(r Result, e error) {
		results++
		if e != nil || r.Text != "hello" {
			t.Fatal(r, e)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if selected != 0 || read != 0 {
		t.Fatal("native operation ran during admission")
	}
	if err := opener.Open(func(Result, error) {}); !errors.Is(err, ErrBusy) {
		t.Fatal("duplicate selection was admitted")
	}
	queued()
	if results != 1 || selected != 1 || read != 1 {
		t.Fatal("selection did not complete exactly once")
	}
	if err := opener.Open(func(Result, error) {}); err != nil {
		t.Fatal("busy state was not released")
	}
	queued()
}

func TestCancelAndNavigationDoNotRead(t *testing.T) {
	for _, mode := range []string{"cancel", "before", "during", "closed"} {
		t.Run(mode, func(t *testing.T) {
			var queued func()
			generation, active := uint64(1), true
			opener := &Opener{Dispatch: func(fn func()) { queued = fn }, Active: func() bool { return active }, Generation: func() uint64 { return generation },
				Select: func() (string, error) {
					if mode == "during" {
						generation++
					}
					if mode == "cancel" {
						return "", nil
					}
					return "chosen", nil
				},
				Read: func(string) (Result, error) { t.Fatal("unapproved or stale selection was read"); return Result{}, nil }}
			called := false
			if err := opener.Open(func(r Result, e error) {
				called = true
				if mode == "cancel" {
					if e != nil || !r.Cancelled {
						t.Fatal(r, e)
					}
				} else if !errors.Is(e, ErrInactive) {
					t.Fatal(e)
				}
			}); err != nil {
				t.Fatal(err)
			}
			if mode == "before" {
				generation++
			}
			if mode == "closed" {
				active = false
			}
			queued()
			if !called {
				t.Fatal("completion missing")
			}
		})
	}
}
