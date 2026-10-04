//go:build windows

package webview2

import (
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

func TestTrayNotificationPayloadAndNoReplay(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	if showTrayNotification(hwnd, "info", "Ready") == nil {
		t.Fatal("accepted notification without an opted-in tray")
	}
	systemTrays.Lock()
	missing := systemTrays.items[hwnd] == nil
	systemTrays.Unlock()
	if !missing {
		t.Fatal("notification installed a tray automatically")
	}
	var submitted []trayIconData
	fail, closing := false, false
	item, err := attachSystemTray(hwnd, strings.Repeat("a", 62)+"\U0001F680", func() bool { return closing }, func(action uint32, data *trayIconData) bool {
		if action == 1 {
			submitted = append(submitted, *data)
			return !fail
		}
		if data.Flags&0x10 != 0 || data.Info[0] != 0 {
			t.Error("registration or cleanup replayed notification")
		}
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	message := strings.Repeat("\uD55C", 253) + "\U0001F680"
	if err := setNativeWindowTitle(hwnd, "Changed caption"); err != nil {
		t.Fatal(err)
	}
	for i, kind := range []string{"info", "warning", "error"} {
		if err := item.showNotification(kind, message); err != nil {
			t.Fatal(err)
		}
		data := submitted[len(submitted)-1]
		if data.Flags != 0x50 || data.InfoFlags != 0x90|uint32(i+1) || windows.UTF16ToString(data.Info[:]) != message || data.Info[255] != 0 || windows.UTF16ToString(data.InfoTitle[:]) != strings.Repeat("a", 62) || data.InfoTitle[63] != 0 {
			t.Fatal("invalid native payload", data)
		}
		if strings.ContainsRune(string(utf16.Decode(data.InfoTitle[:])), '\ufffd') || item.data.Info[0] != 0 || item.data.Flags&0x10 != 0 {
			t.Fatal("invalid UTF-16 or retained balloon")
		}
	}
	if err := item.showNotification("info", "Ready"); err != nil || windows.UTF16ToString(submitted[len(submitted)-1].Info[:]) != "Ready" {
		t.Fatal("short notification retained prior text", err)
	}
	fail = true
	if err := item.showNotification("info", "Ready"); err == nil {
		t.Fatal("accepted shell failure")
	}
	count := len(submitted)
	for _, invalid := range []struct{ kind, message string }{{"other", "Ready"}, {"info", ""}, {"info", "a\x00b"}, {"info", strings.Repeat("a", 256)}} {
		if item.showNotification(invalid.kind, invalid.message) == nil {
			t.Fatal("accepted invalid native request")
		}
	}
	closing = true
	if item.showNotification("info", "Ready") == nil {
		t.Fatal("submitted while closing")
	}
	closing = false
	trayUser32.NewProc("EnableWindow").Call(hwnd, 0)
	if item.showNotification("info", "Ready") == nil {
		t.Fatal("submitted with a disabled modal owner")
	}
	trayUser32.NewProc("EnableWindow").Call(hwnd, 1)
	item.registered = false
	if item.showNotification("info", "Ready") == nil || len(submitted) != count {
		t.Fatal("submitted without registered tray or while unavailable")
	}
	trayUser32.NewProc("SendMessageW").Call(hwnd, trayTaskbarMessage, 0, 0)
	if !item.registered || len(submitted) != count {
		t.Fatal("Explorer recovery replayed notification")
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if item.showNotification("info", "Ready") == nil || item.registered {
		t.Fatal("notification survived destruction")
	}
	if (nativeWindow{}).ShowNotification("info", "Ready") == nil {
		t.Fatal("accepted missing native window")
	}
}

func TestTrayBalloonClickRevealsOnlyLiveOwner(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	closing := false
	item, err := attachSystemTray(hwnd, "Click test", func() bool { return closing }, func(uint32, *trayIconData) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	visible := func() bool { v, _, _ := trayUser32.NewProc("IsWindowVisible").Call(hwnd); return v != 0 }
	click := func(id uintptr) { trayUser32.NewProc("SendMessageW").Call(hwnd, trayMessage, 0, 0x405|id<<16) }
	item.command(trayHide)
	click(2)
	if visible() {
		t.Fatal("wrong icon opened window")
	}
	closing = true
	click(trayIconID)
	if visible() {
		t.Fatal("closing owner reopened")
	}
	closing = false
	trayUser32.NewProc("EnableWindow").Call(hwnd, 0)
	click(trayIconID)
	if visible() {
		t.Fatal("disabled modal owner reopened")
	}
	trayUser32.NewProc("EnableWindow").Call(hwnd, 1)
	click(trayIconID)
	if !visible() {
		t.Fatal("balloon click did not reveal window")
	}
	stateShowWindow.Call(hwnd, showMinimized)
	click(trayIconID)
	if minimized, _, _ := trayUser32.NewProc("IsIconic").Call(hwnd); minimized != 0 {
		t.Fatal("balloon click left window minimized")
	}
}
