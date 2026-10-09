//go:build windows

package webview2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

func TestNativeBrowserFailureClose(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_PROCESS_FAILURE") != "1" {
		t.Skip("opt-in isolated WebView2 browser termination test")
	}
	for _, mode := range []string{"window-close", "system-close"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeBrowserFailureCloseHelper$", "-test.v")
			cmd.Env = append(os.Environ(), "VELOX_PROCESS_FAILURE_CHILD=1", "VELOX_PROCESS_FAILURE_PROFILE="+filepath.Join(t.TempDir(), "profile"), "VELOX_PROCESS_FAILURE_CLOSE="+mode)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("isolated browser-failure close failed: %v (%v)\n%s", err, ctx.Err(), output)
			}
			t.Logf("%s", output)
		})
	}
}

func TestNativeBrowserFailureCloseHelper(t *testing.T) {
	profile := os.Getenv("VELOX_PROCESS_FAILURE_PROFILE")
	if profile == "" || os.Getenv("VELOX_PROCESS_FAILURE_CHILD") != "1" {
		t.Skip("isolated subprocess helper only")
	}
	if _, err := os.Lstat(profile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failure test requires a fresh, nonexisting profile: %v", err)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole32 := windows.NewLazySystemDLL("ole32.dll")
	hr, _, _ := ole32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		t.Fatalf("initialize STA: %08x", hr)
	}
	defer ole32.NewProc("CoUninitialize").Call()
	version, err := webviewloader.GetInstalledVersion()
	if err != nil || version == "" {
		t.Fatalf("installed WebView2 required: %v", err)
	}
	var view WebView
	var operationErr error
	var browser windows.Handle
	var browserPID uint32
	failed := false
	closed := false
	user32 := windows.NewLazySystemDLL("user32.dll")
	options := WebViewOptions{
		DataPath:      profile,
		WindowOptions: WindowOptions{Title: "Velox Isolated Browser Failure Test", Width: 480, Height: 320},
		ShutdownPhase: func(name string) {
			if name == "window-destroyed" {
				closed = true
			}
			if name != "browser-process-exited" {
				return
			}
			failed = true
			if view == nil {
				operationErr = fmt.Errorf("browser failed before initialization")
				return
			}
			message, parameter := uintptr(0x10), uintptr(0)
			if os.Getenv("VELOX_PROCESS_FAILURE_CLOSE") == "system-close" {
				message, parameter = 0x112, 0xf060
			}
			ok, _, err := user32.NewProc("PostMessageW").Call(uintptr(view.Window()), message, parameter, 0)
			if ok == 0 {
				operationErr = fmt.Errorf("post user close: %w", err)
			}
		},
	}
	view, err = NewWithOptionsAndError(options)
	if err != nil || view == nil {
		t.Fatalf("create isolated view: %v", err)
	}
	defer view.Destroy()
	defer func() {
		if browser != 0 {
			_ = windows.CloseHandle(browser)
		}
	}()
	if err := view.Bind("ready", func() {
		view.Dispatch(func() {
			pid, err := view.BrowserProcessID()
			if err != nil || pid == 0 || pid == uint32(os.Getpid()) {
				operationErr = fmt.Errorf("invalid owned browser PID %d: %v", pid, err)
				view.Destroy()
				return
			}
			browserPID = pid
			browser, err = windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, pid)
			if err == nil {
				var path [32768]uint16
				size := uint32(len(path))
				err = windows.QueryFullProcessImageName(browser, 0, &path[0], &size)
				if err == nil && !strings.EqualFold(filepath.Base(windows.UTF16ToString(path[:size])), "msedgewebview2.exe") {
					err = fmt.Errorf("refusing to terminate non-WebView2 process")
				}
			}
			if err == nil {
				err = windows.TerminateProcess(browser, 71)
			}
			if err != nil {
				operationErr = err
				view.Destroy()
			}
		})
	}); err != nil {
		t.Fatal(err)
	}
	view.SetHtml("<html><body>Isolated failure test<script>ready()</script></body></html>")
	view.Run()
	if operationErr != nil || !failed || !closed || !view.(*webview).browserProcessExited {
		t.Fatalf("failed=%v closed=%v error=%v", failed, closed, operationErr)
	}
	if result, err := windows.WaitForSingleObject(browser, 1000); err != nil || result != windows.WAIT_OBJECT_0 {
		t.Fatalf("owned browser did not terminate: %d %v", result, err)
	}
	t.Logf("WebView2=%s Go=%s ownedBrowserPID=%d close=%s: native event and window teardown passed", version, runtime.Version(), browserPID, os.Getenv("VELOX_PROCESS_FAILURE_CLOSE"))
}
