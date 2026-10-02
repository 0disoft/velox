//go:build windows

package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNativeFolderDialogConfiguration(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := dialogOle32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		t.Fatal("STA initialization failed")
	}
	defer dialogOle32.NewProc("CoUninitialize").Call()
	dialog, err := newFolderDialog()
	if err != nil {
		t.Fatal(err)
	}
	defer dialog.call(2)
	var options uint32
	if int32(dialog.call(10, uintptr(unsafe.Pointer(&options)))) < 0 || options&0x02101868 != 0x02101868 || options&0x200 != 0 {
		t.Fatalf("unexpected folder options: %x", options)
	}
}

func TestFolderSnapshotLocalIdentityAndShortName(t *testing.T) {
	root := filepath.Join(t.TempDir(), "long selected directory")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := inspectSelectedFolder(root)
	if err != nil || first.Name != "long selected directory" {
		t.Fatal(first, err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := inspectSelectedFolder(root)
	if err != nil || second.ID != first.ID {
		t.Fatal("content change changed folder identity", err)
	}
	name, _ := windows.UTF16PtrFromString(root)
	var buffer [32768]uint16
	n, err := windows.GetShortPathName(name, &buffer[0], uint32(len(buffer)))
	if err != nil || n == 0 || n >= uint32(len(buffer)) {
		t.Fatal(err)
	}
	alias, err := inspectSelectedFolder(windows.UTF16ToString(buffer[:n]))
	if err != nil || alias != first {
		t.Fatal("short name rejected", alias, err)
	}
}

func TestFolderSnapshotRejectsFilesUnsafePathsAndLinks(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{file, "relative", `\\server\share\folder`, `\\?\C:\folder`, root + ":stream", root + `\..`} {
		if _, err := inspectSelectedFolder(path); !errors.Is(err, ErrFolderUnsupported) {
			t.Fatalf("accepted %q: %v", path, err)
		}
	}
	name, _ := windows.UTF16PtrFromString(root)
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectSelectedFolder(root); !errors.Is(err, ErrFolderUnsupported) {
		t.Fatal("offline folder accepted", err)
	}
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, link); err != nil {
		t.Skip("directory symlink unavailable")
	}
	if _, err := inspectSelectedFolder(link); !errors.Is(err, ErrFolderUnsupported) {
		t.Fatal("folder link accepted", err)
	}
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectSelectedFolder(filepath.Join(link, "child")); !errors.Is(err, ErrFolderUnsupported) {
		t.Fatal("linked parent accepted", err)
	}
	data, _ := os.ReadFile(file)
	if !strings.EqualFold(string(data), "private") {
		t.Fatal("selection modified files")
	}
}
