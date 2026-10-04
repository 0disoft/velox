//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/0disoft/velox/internal/activationkey"
	"golang.org/x/sys/windows"
)

const activationShortcutID = 0x1f20
const hotkeyMessage = 0x312
const modifierNoRepeat = 0x4000

var activationOnce sync.Once
var activationCallback uintptr
var activationOwners = struct {
	sync.Mutex
	items map[uintptr]*activationOwner
}{items: make(map[uintptr]*activationOwner)}

type activationOwner struct {
	binding    activationkey.Binding
	closing    func() bool
	unregister func(uintptr, uint32) bool
	registered bool
}

func registerActivationKey(hwnd uintptr, modifiers, key uint32) bool {
	ok, _, _ := user32Window.NewProc("RegisterHotKey").Call(hwnd, activationShortcutID, uintptr(modifiers), uintptr(key))
	return ok != 0
}

func unregisterActivationKey(hwnd uintptr, id uint32) bool {
	ok, _, _ := user32Window.NewProc("UnregisterHotKey").Call(hwnd, uintptr(id))
	return ok != 0
}

func installActivationShortcut(hwnd uintptr, shortcut activationkey.Shortcut, closing func() bool) error {
	return attachActivationShortcut(hwnd, shortcut, closing, registerActivationKey, unregisterActivationKey)
}

func attachActivationShortcut(hwnd uintptr, shortcut activationkey.Shortcut, closing func() bool, register func(uintptr, uint32, uint32) bool, unregister func(uintptr, uint32) bool) error {
	binding, err := activationkey.Parse(shortcut)
	if err != nil || shortcut == "" {
		return err
	}
	if valid, _, _ := isNativeWindow.Call(hwnd); valid == 0 {
		return errors.New("activation window is unavailable")
	}
	activationOnce.Do(func() { activationCallback = windows.NewCallback(activationWindowProc) })
	owner := &activationOwner{binding: binding, closing: closing, unregister: unregister}
	activationOwners.Lock()
	if activationOwners.items[hwnd] != nil {
		activationOwners.Unlock()
		return errors.New("activation shortcut already installed")
	}
	activationOwners.items[hwnd] = owner
	activationOwners.Unlock()
	if ok, _, _ := permissionComctl32.NewProc("SetWindowSubclass").Call(hwnd, activationCallback, activationShortcutID, 0); ok == 0 {
		dropActivationOwner(hwnd)
		return errors.New("install activation shortcut handler")
	}
	if !register(hwnd, binding.Modifiers|modifierNoRepeat, binding.Key) {
		permissionComctl32.NewProc("RemoveWindowSubclass").Call(hwnd, activationCallback, activationShortcutID)
		dropActivationOwner(hwnd)
		return errors.New("register activation shortcut")
	}
	owner.registered = true
	return nil
}

func dropActivationOwner(hwnd uintptr) {
	activationOwners.Lock()
	delete(activationOwners.items, hwnd)
	activationOwners.Unlock()
}

func (owner *activationOwner) release(hwnd uintptr) {
	if owner.registered {
		owner.registered = false
		owner.unregister(hwnd, activationShortcutID)
	}
}

func activationWindowProc(hwnd, message, wparam, lparam, id, reference uintptr) uintptr {
	activationOwners.Lock()
	owner := activationOwners.items[hwnd]
	activationOwners.Unlock()
	if owner != nil {
		switch message {
		case hotkeyMessage:
			if wparam == activationShortcutID {
				enabled, _, _ := user32Window.NewProc("IsWindowEnabled").Call(hwnd)
				if owner.registered && uint32(lparam&0xffff) == owner.binding.Modifiers && uint32((lparam>>16)&0xffff) == owner.binding.Key && enabled != 0 && (owner.closing == nil || !owner.closing()) {
					revealNativeWindow(hwnd)
				}
				return 0
			}
		case 0x2: // WM_DESTROY.
			owner.release(hwnd)
		case 0x82: // WM_NCDESTROY.
			dropActivationOwner(hwnd)
			owner.release(hwnd)
			permissionComctl32.NewProc("RemoveWindowSubclass").Call(hwnd, activationCallback, activationShortcutID)
		}
	}
	result, _, _ := permissionComctl32.NewProc("DefSubclassProc").Call(hwnd, message, wparam, lparam)
	return result
}

func warnActivationShortcut(hwnd uintptr, shortcut activationkey.Shortcut) {
	text, _ := windows.UTF16PtrFromString(fmt.Sprintf("The activation shortcut %s is unavailable. Another app may already use it.\n\nThis app will continue without a global shortcut.", shortcut))
	title, _ := windows.UTF16PtrFromString("Activation shortcut")
	user32Window.NewProc("MessageBoxW").Call(hwnd, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x30)
}
