package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsLongNameSourceCanSaveToSafeDestination(t *testing.T) {
	root := t.TempDir()
	name := strings.Repeat("\uD55C", 80) + ".txt"
	path := filepath.Join(root, name)
	text := "\uD55C\uAE00\r\noriginal text\r\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := readSelected(path)
	if err != nil || opened.Name != name || opened.Text != text {
		t.Fatalf("long-name read: %+v %v", opened, err)
	}
	if !errors.Is(ValidateSaveName(opened.Name), ErrUnsupported) {
		t.Fatal("native name validation was weakened")
	}
	destination := filepath.Join(root, "Untitled.txt")
	saver := &Saver{
		Dispatch: func(fn func()) { fn() }, Active: func() bool { return true }, Generation: func() uint64 { return 1 },
		Select: func(suggested string) (string, error) {
			if suggested != "Untitled.txt" {
				t.Fatalf("unsafe suggestion: %q", suggested)
			}
			return destination, nil
		},
		Write: writeSelected, Snapshot: snapshotSelected, WriteVersion: writeConnected,
	}
	var result SaveResult
	var saveErr error
	done := func(r SaveResult, err error) { result, saveErr = r, err }
	if err := saver.SaveAs(opened.Text, "Untitled.txt", done); err != nil || saveErr != nil || result.Target == 0 {
		t.Fatalf("safe save: %+v %v %v", result, err, saveErr)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != text {
		t.Fatalf("save bytes changed: %v", err)
	}
	if err := saver.SaveTo("connected edit\r\n", result.Target, done); err != nil || saveErr != nil {
		t.Fatalf("connected save: %v %v", err, saveErr)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "connected edit\r\n" {
		t.Fatalf("connected bytes changed: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != text {
		t.Fatalf("original long-name source changed: %v", err)
	}
}
