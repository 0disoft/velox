package webviewloader

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestLoaderIgnoresAmbientDLL(t *testing.T) {
	if os.Getenv("VELOX_LOADER_TEST_CHILD") == "1" {
		result, err := CompareBrowserVersions("1.0.0.0", "2.0.0.0")
		if err != nil {
			t.Fatalf("embedded version comparison: result=%d err=%v", result, err)
		}
		name, err := windows.UTF16PtrFromString("WebView2Loader.dll")
		if err != nil {
			t.Fatal(err)
		}
		handle, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(uintptr(unsafe.Pointer(name)))
		if handle != 0 {
			t.Fatal("ambient WebView2Loader.dll was loaded instead of the embedded loader")
		}
		if int32(result) >= 0 {
			t.Fatalf("unexpected comparison result: %d", result)
		}
		return
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WebView2Loader.dll"), WebView2Loader, 0600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VELOX_LOADER_TEST_CHILD", "1")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd := exec.Command(exe, "-test.run=^TestLoaderIgnoresAmbientDLL$")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("isolated loader check: %v\n%s", err, output)
	}
}
