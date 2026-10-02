//go:build windows

package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestReadSelectedTextAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want string
		err  error
	}{
		{"unicode", []byte("hello \xed\x95\x9c\xea\xb8\x80"), "hello \xed\x95\x9c\xea\xb8\x80", nil},
		{"bom", []byte("\xef\xbb\xbfhello"), "hello", nil},
		{"empty", []byte{}, "", nil},
		{"invalid", []byte{0xff}, "", ErrUnsupported},
		{"nul", []byte{'a', 0}, "", ErrUnsupported},
		{"exact", []byte(strings.Repeat("a", MaxTextBytes)), strings.Repeat("a", MaxTextBytes), nil},
		{"large", []byte(strings.Repeat("a", MaxTextBytes+1)), "", ErrTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "selected.txt")
			if err := os.WriteFile(path, tc.data, 0o600); err != nil {
				t.Fatal(err)
			}
			result, err := readSelected(path)
			if !errors.Is(err, tc.err) {
				t.Fatalf("read error=%v want %v", err, tc.err)
			}
			if err == nil && (result.Text != tc.want || result.Name != "selected.txt" || result.Bytes != len(tc.data)) {
				t.Fatal("read result differs")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != string(tc.data) {
				t.Fatal("read modified the file")
			}
		})
	}
}

func TestReadRejectsNetworkDevicesStreamsAndDirectories(t *testing.T) {
	for _, path := range []string{`\\server\share\file.txt`, `\\.\pipe\file`, `\\?\C:\file.txt`, `C:\file.txt:stream`, `relative.txt`, `C:file.txt`} {
		if _, err := readSelected(path); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("accepted unsafe path %q: %v", path, err)
		}
	}
	if _, err := readSelected(t.TempDir()); err == nil {
		t.Fatal("read accepted directory")
	}
	path := filepath.Join(t.TempDir(), "offline.txt")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Fatal(err)
	}
	defer windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL)
	if _, err := readSelected(path); !errors.Is(err, ErrUnsupported) {
		t.Fatal("offline file was read")
	}
}

func TestReadRejectsFinalSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "link.txt")
	if err := os.WriteFile(target, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink creation unavailable")
	}
	if _, err := readSelected(link); !errors.Is(err, ErrUnsupported) {
		t.Fatal("symlink was read")
	}
}
