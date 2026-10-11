package webview2

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// Test-only mount-point creation; never invoked for an application profile.
func createStorageProbeJunction(t *testing.T, path, target string) {
	t.Helper()
	if !filepath.IsAbs(path) || !filepath.IsAbs(target) || len(filepath.VolumeName(path)) != 2 || len(filepath.VolumeName(target)) != 2 {
		t.Fatal("junction fixture requires absolute local paths")
	}
	substitute, err := windows.UTF16FromString(`\??\` + filepath.Clean(target))
	if err != nil {
		t.Fatal(err)
	}
	printName, err := windows.UTF16FromString(filepath.Clean(target))
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 16+2*(len(substitute)+len(printName)))
	binary.LittleEndian.PutUint32(data, windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(data[4:], uint16(len(data)-8))
	binary.LittleEndian.PutUint16(data[10:], uint16(2*(len(substitute)-1)))
	binary.LittleEndian.PutUint16(data[12:], uint16(2*len(substitute)))
	binary.LittleEndian.PutUint16(data[14:], uint16(2*(len(printName)-1)))
	for i, value := range append(substitute, printName...) {
		binary.LittleEndian.PutUint16(data[16+i*2:], value)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(browserDataPath(path))
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	var returned uint32
	if err := windows.DeviceIoControl(handle, windows.FSCTL_SET_REPARSE_POINT, &data[0], uint32(len(data)), nil, 0, &returned, nil); err != nil {
		t.Fatalf("set private mount-point fixture: %v", err)
	}
	original, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := os.Stat(path)
	if err != nil || !os.SameFile(original, alias) {
		t.Fatalf("junction did not preserve profile identity: %v", err)
	}
}
