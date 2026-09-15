//go:build windows

package edge

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

func TestNativeFilePermissionRecovery(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_FILE_PERMISSION") != "1" {
		t.Skip("opt-in isolated WebView2 permission test")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := w32.Ole32CoInitializeEx.Call(0, 2)
	if err := hresult(hr); err != nil {
		t.Fatal(err)
	}
	defer windows.NewLazySystemDLL("ole32.dll").NewProc("CoUninitialize").Call()
	class, _ := windows.UTF16PtrFromString("STATIC")
	window, _, _ := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, w32.WSOverlappedWindow, 0, 0, 320, 240, 0, 0, 0, 0)
	if window == 0 {
		t.Fatal("create isolated test window")
	}
	defer w32.User32DestroyWindow.Call(window)
	profilePath := filepath.Join(t.TempDir(), "profile")
	const origin = "https://permission-test.app.invalid"
	const other = "https://unrelated-test.app.invalid"
	for phase := 0; phase < 3; phase++ {
		e := NewChromium()
		e.DataPath = profilePath
		e.FileSystemAccessAllowed = func(uri string) bool { return uri == origin || uri == other }
		if !e.Embed(window) {
			t.Fatal("embed isolated WebView2")
		}
		func() {
			defer e.Destroy()
			if phase == 0 {
				seedNativeFileDenial(t, e, origin)
				seedNativeFileDenial(t, e, other)
			}
			read := func(uri string, reset bool, want CoreWebView2PermissionState) {
				t.Helper()
				completed := false
				var result CoreWebView2PermissionState
				var failure error
				if err := e.FilePermission(uri, reset, func(state CoreWebView2PermissionState, err error) { result, failure, completed = state, err, true }); err != nil {
					t.Fatal(err)
				}
				if !pumpNativeCancellationUntil(10*time.Second, func() bool { return completed }) {
					t.Fatal("permission callback timeout")
				}
				if failure != nil || result != want {
					t.Fatalf("phase=%d reset=%t state=%d want=%d error=%v", phase, reset, result, want, failure)
				}
			}
			if phase < 2 {
				read(origin, false, CoreWebView2PermissionStateDeny)
			}
			if phase == 1 {
				read(origin, true, CoreWebView2PermissionStateDefault)
			}
			if phase == 2 {
				read(origin, false, CoreWebView2PermissionStateDefault)
			}
			read(other, false, CoreWebView2PermissionStateDeny)
			pid, err := e.BrowserProcessID()
			if err != nil {
				t.Fatal(err)
			}
			process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
			if err != nil {
				t.Fatal(err)
			}
			defer windows.CloseHandle(process)
			e.Destroy()
			if !pumpNativeCancellationUntil(12*time.Second, func() bool {
				state, err := windows.WaitForSingleObject(process, 0)
				return err == nil && state == windows.WAIT_OBJECT_0
			}) {
				t.Fatal("isolated browser did not exit")
			}
			if callbackReferenceCount(e) != 0 {
				t.Fatal("permission callback owner leaked")
			}
		}()
	}
	t.Log("persisted denial survived restart; explicit reset verified; reset survived restart; unrelated origin remained denied")
}

func seedNativeFileDenial(t *testing.T, e *Chromium, origin string) {
	t.Helper()
	profile, err := getFilePermissionProfile(e.webview)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseIUnknown(uintptr(unsafe.Pointer(profile)))
	completed := false
	var failure error
	vtbl := &struct {
		_IUnknownVtbl
		Invoke ComProc
	}{
		_IUnknownVtbl{NewComProc(filePermissionQuery), NewComProc(filePermissionAddRef), NewComProc(filePermissionRelease)},
		NewComProc(func(h *filePermissionHandler, hr uintptr) uintptr { failure = hresult(hr); completed = true; return 0 }),
	}
	handler := &filePermissionHandler{vtbl: vtbl, impl: e, iid: filePermissionSetIID}
	var pins runtime.Pinner
	pins.Pin(handler)
	pins.Pin(vtbl)
	defer pins.Unpin()
	uri, _ := windows.UTF16PtrFromString(origin)
	hr, _, _ := profile.vtbl.SetPermissionState.Call(uintptr(unsafe.Pointer(profile)), 8, uintptr(unsafe.Pointer(uri)), 2, uintptr(unsafe.Pointer(handler)))
	runtime.KeepAlive(uri)
	if err := hresult(hr); err != nil {
		t.Fatal(err)
	}
	if !pumpNativeCancellationUntil(10*time.Second, func() bool { return completed }) || failure != nil {
		t.Fatalf("seed isolated denial: %v", failure)
	}
}
