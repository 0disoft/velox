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

func TestSaveNewAndReplaceExactUTF8(t *testing.T) {
	for _, text := range []string{"", "hello \xed\x95\x9c\xea\xb8\x80\n", strings.Repeat("x", MaxTextBytes)} {
		for _, existing := range []bool{false, true} {
			root := t.TempDir()
			path := filepath.Join(root, "notes.md")
			if existing {
				if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			result, err := writeSelected(path, text)
			if err != nil || result.Name != "notes.md" || result.Bytes != len(text) {
				t.Fatal(result, err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != text {
				t.Fatal("saved content differs", err)
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 1 {
				t.Fatal("temporary or backup file leaked", entries)
			}
		}
	}
}

func TestSaveRejectionsPreserveOriginal(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "original.txt")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"a\x00b", string([]byte{0xff}), strings.Repeat("x", MaxTextBytes+1)} {
		if _, err := writeSelected(path, text); err == nil {
			t.Fatal("invalid body saved")
		}
	}
	for _, bad := range []string{root, path + ":stream", `\\server\share\file.txt`, `\\?\C:\file.txt`, "relative.txt"} {
		if _, err := writeSelected(bad, "changed"); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	name, _ := windows.UTF16PtrFromString(path)
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_READONLY); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSelected(path, "changed"); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL); err != nil {
		t.Fatal(err)
	}
	locked, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeSelected(path, "changed"); err == nil {
		t.Fatal("locked file overwritten")
	}
	windows.CloseHandle(locked)
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatal("failed save damaged original")
	}
}

func TestSaveRejectsLinks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "original.txt")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "hard.txt")
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSelected(alias, "changed"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("hard link accepted", err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skip("symlink creation unavailable")
	}
	if _, err := writeSelected(filepath.Join(link, "original.txt"), "changed"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("linked parent accepted", err)
	}
	if _, err := writeSelected(filepath.Join(link, "new.txt"), "changed"); !errors.Is(err, ErrUnsupported) {
		t.Fatal("linked parent create accepted", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "keep" {
		t.Fatal("link save modified original")
	}
}

func TestSavePreservesRestrictedDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	before, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeSelected(path, "new private text"); err != nil {
		t.Fatal(err)
	}
	after, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil || before.String() != after.String() {
		t.Fatal("replacement changed DACL", before.String(), after, err)
	}
}

func TestSaveThroughShortPathAlias(t *testing.T) {
	root := filepath.Join(t.TempDir(), "long directory name for saving")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(root)
	var buffer [32768]uint16
	n, err := windows.GetShortPathName(name, &buffer[0], uint32(len(buffer)))
	if err != nil || n == 0 || n >= uint32(len(buffer)) {
		t.Fatal("short path lookup failed", err)
	}
	shortRoot := windows.UTF16ToString(buffer[:n])
	if strings.EqualFold(shortRoot, root) {
		t.Skip("volume does not generate short path aliases")
	}
	path := filepath.Join(shortRoot, "notes.txt")
	if _, err := writeSelected(path, "first"); err != nil {
		t.Fatal("new save through short alias", err)
	}
	version, err := snapshotSelected(path)
	if err != nil {
		t.Fatal("snapshot through short alias", err)
	}
	if _, err := writeConnected(path, "second", version); err != nil {
		t.Fatal("connected save through short alias", err)
	}
	if _, err := writeSelected(path, "third"); err != nil {
		t.Fatal("replacement through short alias", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil || string(data) != "third" {
		t.Fatal("long path readback differs", err)
	}
}
