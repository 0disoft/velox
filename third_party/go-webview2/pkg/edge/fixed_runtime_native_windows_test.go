//go:build windows

package edge

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

// Opt-in: use a private, verified Fixed Version folder rather than replacing
// Evergreen or sharing an existing application's browser profile.
func TestNativeFixedRuntimeCompatibility(t *testing.T) {
	folder := os.Getenv("VELOX_NATIVE_FIXED_RUNTIME")
	if folder == "" {
		t.Skip("opt-in Fixed Version runtime compatibility witness")
	}
	expectedVersion := os.Getenv("VELOX_NATIVE_FIXED_VERSION")
	expectedSettings := os.Getenv("VELOX_NATIVE_FIXED_SETTINGS8")
	if expectedVersion == "" || expectedSettings != "available" && expectedSettings != "unavailable" {
		t.Fatal("fixed runtime requires an expected version and available/unavailable Settings8 expectation")
	}
	selectedVersion, err := webviewloader.GetAvailableVersion(folder)
	if err != nil || selectedVersion != expectedVersion {
		t.Fatalf("loader-selected version=%q, want %q: %v", selectedVersion, expectedVersion, err)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := w32.Ole32CoInitializeEx.Call(0, 2)
	if err := hresult(hr); err != nil {
		t.Fatal(err)
	}
	defer windows.NewLazySystemDLL("ole32.dll").NewProc("CoUninitialize").Call()
	class, _ := windows.UTF16PtrFromString("STATIC")
	window, _, err := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, w32.WSOverlappedWindow, 0, 0, 320, 240, 0, 0, 0, 0)
	if window == 0 {
		t.Fatal(err)
	}
	defer w32.User32DestroyWindow.Call(window)
	e := NewChromium()
	e.DataPath = filepath.Join(t.TempDir(), "profile")
	e.BrowserExecutableFolder = folder
	e.DenyFrames, e.DenyNewWindows, e.DenyDownloads = true, true, true
	e.NavigationAllowed = func(uri string) bool { return strings.HasPrefix(uri, "https://fixture.app.invalid/") }
	defer e.Destroy()
	if !e.Embed(window) {
		t.Fatalf("initialize selected runtime: %v", e.InitializationError())
	}
	pid, err := e.BrowserProcessID()
	if err != nil {
		t.Fatal(err)
	}
	browser, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(browser)
	defer func() {
		e.Destroy()
		state, err := windows.WaitForSingleObject(browser, 10000)
		if err != nil || state != windows.WAIT_OBJECT_0 {
			t.Errorf("selected browser did not exit: state=%x error=%v", state, err)
		}
	}()
	var image [32768]uint16
	length := uint32(len(image))
	if err := windows.QueryFullProcessImageName(browser, 0, &image[0], &length); err != nil {
		t.Fatal(err)
	}
	actual, err := filepath.EvalSymlinks(windows.UTF16ToString(image[:length]))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(filepath.Join(folder, "msedgewebview2.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(actual, expected) {
		t.Fatalf("runtime selection mismatch: actual=%s expected=%s", actual, expected)
	}
	var version *uint16
	hr, _, _ = e.environment.vtbl.GetBrowserVersionString.Call(uintptr(unsafe.Pointer(e.environment)), uintptr(unsafe.Pointer(&version)))
	if err := hresult(hr); err != nil {
		t.Fatal(err)
	}
	if version == nil {
		t.Fatal("selected runtime returned no version")
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(version))
	actualVersion := windows.UTF16PtrToString(version)
	if actualVersion != expectedVersion {
		t.Fatalf("runtime version=%q, want %q", actualVersion, expectedVersion)
	}
	settings, err := e.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	supported, err := settings.SetReputationCheckingRequired(false)
	settings.Release()
	if err != nil || supported != (expectedSettings == "available") {
		t.Fatalf("Settings8: supported=%v expected=%s error=%v", supported, expectedSettings, err)
	}
	t.Logf("selected runtime verified: version=%s executable=%s Settings8=%v", actualVersion, actual, supported)
}
