//go:build windows

package inspector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/artifactlimits"
	"golang.org/x/sys/windows"
)

func TestInspectDirectoryRejectsOversizedEntryBeforeOpeningIt(t *testing.T) {
	build := buildFixture(t)
	path := filepath.Join(build.DirectoryPath, "aaa-oversized.bin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	var returned uint32
	if err := windows.DeviceIoControl(windows.Handle(file.Fd()), windows.FSCTL_SET_SPARSE, nil, 0, nil, 0, &returned, nil); err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(artifactlimits.MaxEntryBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	// Metadata stays readable, but opening file contents must fail. This also
	// prevents a regression from hashing a 512 MiB sparse fixture in the test.
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	if opened, err := os.Open(path); err == nil {
		opened.Close()
		t.Fatal("fixture unexpectedly allows content reads")
	}
	if _, err := Inspect(build.DirectoryPath); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("Inspect() = %v, want size rejection before opening contents", err)
	}
}
