//go:build windows

package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

const nativePhasePrefix = "VELOX_NATIVE_PHASE_RESULT="

type nativePhasePoint struct {
	Name string  `json:"name"`
	MS   float64 `json:"elapsedMs"`
}

type nativePhaseResult struct {
	Success                bool               `json:"success"`
	BrowserPID             uint32             `json:"browserPid"`
	ControllerCalls        int                `json:"controllerCalls"`
	ControllerHRESULT      string             `json:"controllerHresult"`
	EnvironmentMS          float64            `json:"environmentMs"`
	ControllerEntryMS      float64            `json:"controllerEntryMs"`
	ControllerSetupMS      float64            `json:"controllerSetupMs"`
	EmbedMS                float64            `json:"embedMs"`
	DestroyMS              float64            `json:"destroyMs"`
	CallbackReferences     uintptr            `json:"callbackReferencesAfterDestroy"`
	OwnedInterfacesCleared bool               `json:"ownedInterfacesCleared"`
	Shutdown               []nativePhasePoint `json:"shutdown"`
}

type nativePhaseObserver struct {
	*Chromium
	started time.Time
	result  *nativePhaseResult
}

func (o *nativePhaseObserver) CreateCoreWebView2ControllerCompleted(hr uintptr, controller *ICoreWebView2Controller) uintptr {
	o.result.ControllerCalls++
	o.result.ControllerEntryMS = float64(time.Since(o.started)) / float64(time.Millisecond)
	o.result.ControllerHRESULT = fmt.Sprintf("0x%08x", hr)
	return o.Chromium.CreateCoreWebView2ControllerCompleted(hr, controller)
}

func TestNativeRelaunchPhases(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_RELAUNCH_PHASES") != "1" {
		t.Skip("opt-in source-fork relaunch phase diagnostic")
	}
	version, err := webviewloader.GetInstalledVersion()
	if err != nil || version == "" {
		t.Fatalf("installed WebView2 is required: %v", err)
	}
	t.Logf("source-fork initialization-only evidence: WebView2=%s Go=%s arch=%s; one same-profile then one fresh-profile pair; not a release-host or DOM-ready benchmark", version, runtime.Version(), runtime.GOARCH)
	for _, mode := range []string{"same-profile", "fresh-profile"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			firstProfile := filepath.Join(root, "first")
			secondProfile := firstProfile
			if mode == "fresh-profile" {
				secondProfile = filepath.Join(root, "second")
			}
			first := runNativePhaseChild(t, firstProfile)
			second := runNativePhaseChild(t, secondProfile)
			t.Logf("environment-to-callback-ms first=%.3f second=%.3f; callback-to-setup-ms first=%.3f second=%.3f", first.ControllerEntryMS-first.EnvironmentMS, second.ControllerEntryMS-second.EnvironmentMS, first.ControllerSetupMS-first.ControllerEntryMS, second.ControllerSetupMS-second.ControllerEntryMS)
		})
	}
}

func runNativePhaseChild(t *testing.T, profile string) nativePhaseResult {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestNativeRelaunchPhaseChild$", "-test.v")
	cmd.Env = append(os.Environ(), "VELOX_NATIVE_PHASE_CHILD_PROFILE="+profile)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, runErr := cmd.CombinedOutput()
	t.Logf("child output:\n%s", output)
	var result nativePhaseResult
	found := false
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, nativePhasePrefix) {
			continue
		}
		if found {
			t.Fatal("duplicate phase result")
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, nativePhasePrefix))), &result); err != nil {
			t.Fatal(err)
		}
		found = true
	}
	if result.BrowserPID != 0 {
		browser, err := windows.OpenProcess(windows.SYNCHRONIZE, false, result.BrowserPID)
		if err == nil {
			// Wait only during cleanup, after both launches, never before relaunch.
			t.Cleanup(func() {
				defer windows.CloseHandle(browser)
				state, err := windows.WaitForSingleObject(browser, 10000)
				if err != nil || state != windows.WAIT_OBJECT_0 {
					t.Errorf("browser cleanup: state=0x%08x error=%v", state, err)
				}
			})
		} else if err != windows.ERROR_INVALID_PARAMETER {
			t.Errorf("open browser cleanup handle: %v", err)
		}
	}
	if runErr != nil || !found || !result.Success {
		t.Fatalf("native phase child failed: process=%v result=%+v", runErr, result)
	}
	return result
}

func TestNativeRelaunchPhaseChild(t *testing.T) {
	profile := os.Getenv("VELOX_NATIVE_PHASE_CHILD_PROFILE")
	if profile == "" {
		t.Skip("private child of opt-in phase diagnostic")
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
	e.DataPath = profile
	result := nativePhaseResult{}
	defer func() {
		e.Destroy()
		data, err := json.Marshal(result)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(nativePhasePrefix + string(data))
	}()
	started := time.Now()
	observer := &nativePhaseObserver{Chromium: e, started: started, result: &result}
	e.controllerCompleted.impl = observer
	e.StartupPhase = func(name string) {
		ms := float64(time.Since(started)) / float64(time.Millisecond)
		switch name {
		case "environment-created":
			result.EnvironmentMS = ms
		case "controller-created":
			result.ControllerSetupMS = ms
		}
	}
	if !e.Embed(window) {
		t.Fatalf("Embed: %v", e.initializationError)
	}
	result.EmbedMS = float64(time.Since(started)) / float64(time.Millisecond)
	result.BrowserPID, err = e.BrowserProcessID()
	if err != nil {
		t.Fatal(err)
	}
	destroyStarted := time.Now()
	e.ShutdownPhase = func(name string) {
		result.Shutdown = append(result.Shutdown, nativePhasePoint{Name: name, MS: float64(time.Since(destroyStarted)) / float64(time.Millisecond)})
	}
	e.Destroy()
	result.DestroyMS = float64(time.Since(destroyStarted)) / float64(time.Millisecond)
	result.CallbackReferences = callbackReferenceCount(e)
	result.OwnedInterfacesCleared = e.webview == nil && e.controller == nil && e.environment == nil
	if result.ControllerCalls != 1 || result.ControllerHRESULT != "0x00000000" || !result.OwnedInterfacesCleared || result.EnvironmentMS > result.ControllerEntryMS || result.ControllerEntryMS > result.ControllerSetupMS || result.ControllerSetupMS > result.EmbedMS {
		t.Fatalf("invalid native phase result: %+v", result)
	}
	result.Success = true
}
