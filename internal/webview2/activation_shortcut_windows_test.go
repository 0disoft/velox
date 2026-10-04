//go:build windows

package webview2

import (
	"testing"

	"github.com/0disoft/velox/internal/activationkey"
	"golang.org/x/sys/windows"
)

func TestActivationShortcutOffAndRegistrationRollback(t *testing.T) {
	if installActivationShortcut(0, "", nil) != nil {
		t.Fatal("disabled shortcut touched native window")
	}
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	registers, unregisters := 0, 0
	register := func(uintptr, uint32, uint32) bool { registers++; return false }
	unregister := func(uintptr, uint32) bool { unregisters++; return true }
	if attachActivationShortcut(hwnd, "Ctrl+V", nil, register, unregister) == nil || registers != 0 {
		t.Fatal("invalid shortcut reached native API")
	}
	if attachActivationShortcut(0, "Ctrl+Alt+V", nil, register, unregister) == nil || registers != 0 {
		t.Fatal("invalid window reached registration")
	}
	if attachActivationShortcut(hwnd, "Ctrl+Alt+V", nil, register, unregister) == nil || registers != 1 || unregisters != 0 {
		t.Fatal("registration failure removed an unowned hotkey")
	}
	activationOwners.Lock()
	remaining := activationOwners.items[hwnd]
	activationOwners.Unlock()
	if remaining != nil {
		t.Fatal("registration failure retained handler owner")
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if unregisters != 0 {
		t.Fatal("failed registration left cleanup attached")
	}
}

func TestActivationShortcutMessagesAndLifetime(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	closing := false
	registers, unregisters := 0, 0
	register := func(h uintptr, mods, key uint32) bool {
		registers++
		if h != hwnd || mods != 7|modifierNoRepeat || key != 'V' {
			t.Fatal(h, mods, key)
		}
		return true
	}
	unregister := func(h uintptr, id uint32) bool {
		unregisters++
		if h != hwnd || id != activationShortcutID {
			t.Fatal(h, id)
		}
		// Simulate native message reentry while releasing.
		activationWindowProc(hwnd, 0x2, 0, 0, activationShortcutID, 0)
		return true
	}
	if err := attachActivationShortcut(hwnd, "Ctrl+Alt+Shift+V", func() bool { return closing }, register, unregister); err != nil {
		t.Fatal(err)
	}
	if attachActivationShortcut(hwnd, "Ctrl+Alt+Shift+V", nil, register, unregister) == nil || registers != 1 {
		t.Fatal("duplicate install replaced the binding")
	}
	visible := func() bool { v, _, _ := user32Window.NewProc("IsWindowVisible").Call(hwnd); return v != 0 }
	send := func(id uintptr, mods, key uint32) {
		user32Window.NewProc("SendMessageW").Call(hwnd, hotkeyMessage, id, uintptr(mods)|uintptr(key)<<16)
	}
	showWindow.Call(hwnd, 0)
	send(activationShortcutID+1, 7, 'V')
	send(activationShortcutID, 3, 'V')
	send(activationShortcutID, 7, 'B')
	if visible() {
		t.Fatal("unmatched hotkey revealed window")
	}
	closing = true
	send(activationShortcutID, 7, 'V')
	if visible() {
		t.Fatal("closing owner reopened")
	}
	closing = false
	user32Window.NewProc("EnableWindow").Call(hwnd, 0)
	send(activationShortcutID, 7, 'V')
	if visible() {
		t.Fatal("disabled modal owner reopened")
	}
	user32Window.NewProc("EnableWindow").Call(hwnd, 1)
	send(activationShortcutID, 7, 'V')
	if !visible() {
		t.Fatal("hotkey did not reveal owner without a tray")
	}
	showWindow.Call(hwnd, showMaximized)
	showWindow.Call(hwnd, 0)
	send(activationShortcutID, 7, 'V')
	if maximized, _, _ := isZoomed.Call(hwnd); maximized == 0 {
		t.Fatal("hotkey discarded maximized placement")
	}
	showWindow.Call(hwnd, showMinimized)
	send(activationShortcutID, 7, 'V')
	if minimized, _, _ := isIconic.Call(hwnd); minimized != 0 {
		t.Fatal("hotkey left owner minimized")
	}
	guard := windows.NewCallback(func(h, m, w, l, id, ref uintptr) uintptr {
		if m == 0x10 { // A cancelled normal WM_CLOSE must retain the binding.
			return 0
		}
		r, _, _ := permissionComctl32.NewProc("DefSubclassProc").Call(h, m, w, l)
		return r
	})
	if ok, _, _ := permissionComctl32.NewProc("SetWindowSubclass").Call(hwnd, guard, activationShortcutID+1, 0); ok == 0 {
		t.Fatal("install cancelled-close guard")
	}
	user32Window.NewProc("SendMessageW").Call(hwnd, 0x10, 0, 0)
	activationOwners.Lock()
	owner := activationOwners.items[hwnd]
	activationOwners.Unlock()
	if owner == nil || !owner.registered || unregisters != 0 {
		t.Fatal("cancelled close removed activation shortcut")
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	activationOwners.Lock()
	remaining := activationOwners.items[hwnd]
	activationOwners.Unlock()
	if remaining != nil || unregisters != 1 {
		t.Fatal("destruction did not clean up exactly once", remaining, unregisters)
	}
}

func TestActivationShortcutRealRegistrationCollisionAndRelease(t *testing.T) {
	stateTestThread(t)
	first, second := stateTestWindow(t), stateTestWindow(t)
	var chosen activationkey.Shortcut
	for _, key := range "9876543210ZYXWVUTSRQPONMLKJIHGFEDCBA" {
		candidate := activationkey.Shortcut("Ctrl+Alt+Shift+" + string(key))
		if installActivationShortcut(first, candidate, nil) == nil {
			chosen = candidate
			break
		}
	}
	if chosen == "" {
		t.Skip("no free Ctrl+Alt+Shift shortcut on this desktop")
	}
	t.Logf("real Windows registration/collision/release: %s", chosen)
	if installActivationShortcut(second, chosen, nil) == nil {
		t.Fatal("second window stole registered shortcut")
	}
	// A failed second registration must leave the first owner's shortcut intact.
	binding, _ := activationkey.Parse(chosen)
	if registerActivationKey(second, binding.Modifiers|modifierNoRepeat, binding.Key) {
		unregisterActivationKey(second, activationShortcutID)
		t.Fatal("failed collision unregistered the original shortcut")
	}
	stateUser32.NewProc("DestroyWindow").Call(first)
	if err := installActivationShortcut(second, chosen, nil); err != nil {
		t.Fatal("destroy did not release shortcut for reuse", err)
	}
	stateUser32.NewProc("DestroyWindow").Call(second)
}
