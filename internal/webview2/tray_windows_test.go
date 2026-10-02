//go:build windows

package webview2

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestTrayLayoutAndTooltip(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) == 8 && (unsafe.Sizeof(trayIconData{}) != 976 || unsafe.Offsetof(trayIconData{}.Window) != 8 || unsafe.Offsetof(trayIconData{}.Icon) != 32 || unsafe.Offsetof(trayIconData{}.BalloonIcon) != 968) {
		t.Fatal("NOTIFYICONDATAW ABI drift")
	}
	for _, input := range []string{"Velox", "\uD55C\uAE00", strings.Repeat("a", 126) + "\U0001F680", "a\x00b"} {
		text := trayTooltip(input)
		if len(text) > 127 || strings.ContainsRune(string(utf16.Decode(text)), '\ufffd') || strings.ContainsRune(string(utf16.Decode(text)), 0) {
			t.Fatalf("unsafe tooltip: %v", text)
		}
	}
	if err := installSystemTray(0, "Disabled", false, nil); err != nil {
		t.Fatalf("disabled tray touched native window: %v", err)
	}
}

func TestTrayRegistrationRollback(t *testing.T) {
	stateTestThread(t)
	for _, fail := range []uint32{0, 4} {
		hwnd := stateTestWindow(t)
		var calls []uint32
		_, err := attachSystemTray(hwnd, "Rollback", nil, func(action uint32, data *trayIconData) bool {
			calls = append(calls, action)
			return action != fail
		})
		if err == nil {
			t.Fatal("accepted failed registration")
		}
		want := []uint32{0}
		if fail == 4 {
			want = []uint32{0, 4, 2}
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("calls=%v, want %v", calls, want)
		}
		systemTrays.Lock()
		remaining := systemTrays.items[hwnd]
		systemTrays.Unlock()
		if remaining != nil {
			t.Fatal("failed registration retained owner")
		}
		stateUser32.NewProc("DestroyWindow").Call(hwnd)
		if !reflect.DeepEqual(calls, want) {
			t.Fatal("failed subclass remained attached")
		}
	}
}

func TestTrayShellRegistrationSmoke(t *testing.T) {
	stateTestThread(t)
	class, _ := windows.UTF16PtrFromString("Shell_TrayWnd")
	shell, _, _ := trayUser32.NewProc("FindWindowW").Call(uintptr(unsafe.Pointer(class)), 0)
	if shell == 0 {
		t.Skip("Windows notification area is not running")
	}
	hwnd := stateTestWindow(t)
	item, err := attachSystemTray(hwnd, "Velox disposable tray test", nil, shellNotifyIcon)
	if err != nil {
		t.Fatal(err)
	}
	if !item.registered {
		t.Fatal("shell did not register icon")
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if item.registered {
		t.Fatal("shell icon remained registered after destruction")
	}
}

func TestTrayNativeCommandsRecoveryAndCleanup(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	closing, failAdd := false, false
	var calls []uint32
	item, err := attachSystemTray(hwnd, "Native test", func() bool { return closing }, func(action uint32, data *trayIconData) bool {
		calls = append(calls, action)
		if data.Window != hwnd || data.Version != 4 || data.Icon == 0 || data.Callback != trayMessage {
			t.Error("invalid notification data")
		}
		return action != 0 || !failAdd
	})
	if err != nil {
		t.Fatal(err)
	}
	send := func(message, wp, lp uintptr) { trayUser32.NewProc("SendMessageW").Call(hwnd, message, wp, lp) }
	visible := func() bool { v, _, _ := trayUser32.NewProc("IsWindowVisible").Call(hwnd); return v != 0 }
	item.command(trayShow)
	item.command(trayHide)
	if visible() {
		t.Fatal("hide left window visible")
	}
	send(trayMessage, 0, 0x400|2<<16) // Wrong icon ID must not activate.
	if visible() {
		t.Fatal("accepted wrong tray icon identity")
	}
	send(trayMessage, 0, 0x401|trayIconID<<16)
	if !visible() {
		t.Fatal("keyboard activation left window hidden")
	}
	stateShowWindow.Call(hwnd, showMaximized)
	item.command(trayHide)
	item.command(trayShow)
	maximized, _, _ := trayUser32.NewProc("IsZoomed").Call(hwnd)
	if maximized == 0 {
		t.Fatal("show discarded maximized placement")
	}
	stateShowWindow.Call(hwnd, showMinimized)
	item.command(trayShow)
	minimized, _, _ := trayUser32.NewProc("IsIconic").Call(hwnd)
	if minimized != 0 {
		t.Fatal("show left window minimized")
	}
	closing = true
	item.command(trayHide)
	if !visible() {
		t.Fatal("acted while closing")
	}
	closing = false
	trayUser32.NewProc("EnableWindow").Call(hwnd, 0)
	item.command(trayHide)
	if !visible() {
		t.Fatal("hid disabled modal owner")
	}
	send(trayTaskbarMessage, 0, 0) // Re-register even when a modal disables its owner.
	trayUser32.NewProc("EnableWindow").Call(hwnd, 1)
	item.command(trayHide)
	failAdd = true
	send(trayTaskbarMessage, 0, 0)
	if !visible() || item.registered {
		t.Fatal("failed restart stranded hidden window")
	}
	item.command(trayHide)
	if !visible() {
		t.Fatal("hid window without a recoverable tray icon")
	}
	failAdd = false
	send(trayTaskbarMessage, 0, 0)
	if !item.registered {
		t.Fatal("did not restore tray")
	}
	item.command(trayHide)
	// A cancelled normal close must retain the icon and owner.
	closeCount := 0
	guard := windows.NewCallback(func(h, m, w, l, id, ref uintptr) uintptr {
		if m == 0x10 {
			closeCount++
			return 0
		}
		r, _, _ := permissionComctl32.NewProc("DefSubclassProc").Call(h, m, w, l)
		return r
	})
	permissionComctl32.NewProc("SetWindowSubclass").Call(hwnd, guard, 0x1f14, 0)
	item.command(trayQuit)
	type message struct {
		Window         uintptr
		ID             uint32
		Wparam, Lparam uintptr
		Time           uint32
		X, Y           int32
		Private        uint32
	}
	var msg message
	got, _, _ := trayUser32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&msg)), hwnd, 0x10, 0x10, 1)
	if got == 0 || msg.Wparam != 0 || msg.Lparam != 0 || !visible() {
		t.Fatal("quit bypassed normal visible-owner close")
	}
	trayUser32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	if closeCount != 1 || !item.registered {
		t.Fatal("cancelled close lost icon")
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	systemTrays.Lock()
	remaining := systemTrays.items[hwnd]
	systemTrays.Unlock()
	if remaining != nil || item.registered || calls[len(calls)-1] != 2 {
		t.Fatal("destroy retained tray owner or icon")
	}
	deleteCount := 0
	for _, action := range calls {
		if action == 2 {
			deleteCount++
		}
	}
	if deleteCount != 1 {
		t.Fatalf("delete count=%d", deleteCount)
	}
}
