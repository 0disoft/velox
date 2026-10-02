//go:build windows

package singleinstance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestSingleInstanceHelper(t *testing.T) {
	if os.Getenv("VELOX_INSTANCE_TEST_HELPER") != "1" {
		return
	}
	g, primary, err := Acquire("dev.velox.instance-test", os.Getenv("VELOX_INSTANCE_TEST_PROFILE"))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	fmt.Printf("primary=%t\n", primary)
	if os.Getenv("VELOX_INSTANCE_TEST_NO_CLOSE") == "1" {
		os.Exit(0)
	}
}

func TestSingleInstanceCrossProcessAndRelease(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "not-created")
	probe := func(want bool) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSingleInstanceHelper$")
		cmd.Env = append(os.Environ(), "VELOX_INSTANCE_TEST_HELPER=1", "VELOX_INSTANCE_TEST_PROFILE="+profile, "VELOX_INSTANCE_TEST_NO_CLOSE=1")
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), fmt.Sprintf("primary=%t", want)) {
			t.Fatalf("probe want %t: %s, %v", want, out, err)
		}
	}
	g, primary, err := Acquire("dev.velox.instance-test", profile)
	if err != nil || !primary {
		t.Fatalf("primary=%t, %v", primary, err)
	}
	t.Cleanup(g.Close)
	var flags uint32
	result, _, handleErr := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetHandleInformation").Call(uintptr(g.handle), uintptr(unsafe.Pointer(&flags)))
	if result == 0 || flags&windows.HANDLE_FLAG_INHERIT != 0 {
		t.Fatalf("mutex handle must not be inherited: flags=%d, %v", flags, handleErr)
	}
	probe(false)
	// A duplicate while initializing is suppressed without creating a profile.
	duplicate, primary, err := Acquire("dev.velox.instance-test", strings.ToUpper(profile))
	if err != nil || primary {
		t.Fatalf("case alias primary=%t, %v", primary, err)
	}
	duplicate.Activate()
	duplicate.Close()
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("created profile: %v", err)
	}
	for _, key := range []struct{ app, path string }{
		{"dev.velox.other", profile}, {"dev.velox.instance-test", profile + "-other"},
	} {
		other, primary, err := Acquire(key.app, key.path)
		if err != nil || !primary {
			t.Fatalf("independent primary=%t, %v", primary, err)
		}
		other.Close()
	}
	g.Close()
	probe(true)
	probe(true) // The helper's process exit releases its own lease as well.
}

func TestCanonicalProfileResolvesExistingAncestor(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "missing", "tail")
	before, err := canonicalProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	after, err := canonicalProfile(profile)
	if err != nil || before != after {
		t.Fatalf("path drift %q -> %q, %v", before, after, err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(filepath.Join(root, "missing"), alias); err != nil {
		t.Logf("directory alias test unavailable: %v", err)
	} else if got, err := canonicalProfile(filepath.Join(alias, "tail")); err != nil || got != after {
		t.Fatalf("alias=%q, want %q, %v", got, after, err)
	}
	if _, err := canonicalProfile("relative"); err == nil {
		t.Fatal("accepted relative profile")
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalProfile(filepath.Join(file, "tail")); err == nil {
		t.Fatal("accepted file ancestor")
	}
}

func TestSingleInstanceNativeWindowActivationAndCleanup(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	g, primary, err := Acquire("dev.velox.native-test", t.TempDir())
	if err != nil || !primary {
		t.Fatalf("primary=%t, %v", primary, err)
	}
	defer g.Close()
	class, _ := windows.UTF16PtrFromString("STATIC")
	title, _ := windows.UTF16PtrFromString("Velox single-instance test")
	hwnd, _, err := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0xcf0000, 100, 100, 400, 300, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("create disposable window: %v", err)
	}
	destroy := user32.NewProc("DestroyWindow")
	defer destroy.Call(hwnd)
	if err := g.Attach(hwnd); err != nil {
		t.Fatal(err)
	}
	showWindow.Call(hwnd, 6) // SW_MINIMIZE
	g.Activate()
	type message struct {
		Window         uintptr
		ID             uint32
		Wparam, Lparam uintptr
		Time           uint32
		X, Y           int32
		Private        uint32
	}
	var msg message
	got, _, _ := user32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&msg)), hwnd, activationMessage, activationMessage, 1)
	if got == 0 || msg.Wparam != 0 || msg.Lparam != 0 {
		t.Fatalf("activation message = %+v", msg)
	}
	user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	minimized, _, _ := isIconic.Call(hwnd)
	if minimized != 0 {
		t.Fatal("duplicate activation left window minimized")
	}
	showWindow.Call(hwnd, 0) // SW_HIDE, as used by an opt-in tray.
	user32.NewProc("SendMessageW").Call(hwnd, activationMessage, 0, 0)
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(hwnd)
	if visible == 0 {
		t.Fatal("duplicate activation left window hidden")
	}
	destroy.Call(hwnd)
	owners.Lock()
	remaining := owners.items[hwnd]
	owners.Unlock()
	if remaining != nil {
		t.Fatal("destroy retained window owner")
	}
	if err := g.Attach(0); err == nil {
		t.Fatal("accepted invalid window")
	}
}
