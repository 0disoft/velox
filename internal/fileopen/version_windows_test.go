//go:build windows

package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConnectedSaveRefreshesVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if _, err := writeSelected(path, "first"); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"second", "third"} {
		version, err := snapshotSelected(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writeConnected(path, text, version); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != text {
			t.Fatal("write differs", err)
		}
	}
}

func TestConnectedSaveRejectsExternalChanges(t *testing.T) {
	for _, mode := range []string{"content", "same-size-and-time", "replacement", "deleted"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "notes.txt")
			if _, err := writeSelected(path, "first"); err != nil {
				t.Fatal(err)
			}
			version, err := snapshotSelected(path)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "replacement" || mode == "deleted" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			if mode != "deleted" {
				text := "other"
				if mode == "replacement" {
					text = "first"
				}
				if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
					t.Fatal(err)
				}
				if mode != "content" {
					if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
						t.Fatal(err)
					}
				}
			}
			if _, err := writeConnected(path, "do not overwrite", version); !errors.Is(err, ErrConflict) {
				t.Fatal("change not rejected", err)
			}
			if mode == "deleted" {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("deleted target was recreated")
				}
			} else {
				data, err := os.ReadFile(path)
				want := "other"
				if mode == "replacement" {
					want = "first"
				}
				if err != nil || string(data) != want {
					t.Fatal("external content was overwritten")
				}
			}
		})
	}
}
