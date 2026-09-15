//go:build windows

package webview2

import (
	"fmt"
	"sync"
	"unsafe"

	webview "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const filePermissionMenuID = 0x1f10

var permissionUser32 = windows.NewLazySystemDLL("user32.dll")
var permissionComctl32 = windows.NewLazySystemDLL("comctl32.dll")
var permissionMenus = struct {
	sync.Mutex
	items map[uintptr]*filePermissionMenu
}{items: make(map[uintptr]*filePermissionMenu)}
var permissionSubclass uintptr

func init() { permissionSubclass = windows.NewCallback(filePermissionWindowProc) }

type filePermissionMenu struct {
	view   webview.WebView
	hwnd   uintptr
	origin string
	busy   bool
}

func installFilePermissionMenu(view webview.WebView, origin string) error {
	hwnd := uintptr(view.Window())
	menu, _, _ := permissionUser32.NewProc("GetSystemMenu").Call(hwnd, 0)
	if menu == 0 {
		return fmt.Errorf("get window system menu")
	}
	label, _ := windows.UTF16PtrFromString("&File access...")
	result, _, _ := permissionUser32.NewProc("AppendMenuW").Call(menu, 0, filePermissionMenuID, uintptr(unsafe.Pointer(label)))
	if result == 0 {
		return fmt.Errorf("add file access menu")
	}
	item := &filePermissionMenu{view: view, hwnd: hwnd, origin: origin}
	permissionMenus.Lock()
	permissionMenus.items[hwnd] = item
	permissionMenus.Unlock()
	result, _, _ = permissionComctl32.NewProc("SetWindowSubclass").Call(hwnd, permissionSubclass, filePermissionMenuID, 0)
	if result == 0 {
		permissionMenus.Lock()
		delete(permissionMenus.items, hwnd)
		permissionMenus.Unlock()
		permissionUser32.NewProc("RemoveMenu").Call(menu, filePermissionMenuID, 0)
		return fmt.Errorf("install file access window handler")
	}
	return nil
}

func filePermissionWindowProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	permissionMenus.Lock()
	item := permissionMenus.items[hwnd]
	if message == 0x82 {
		delete(permissionMenus.items, hwnd)
	} // WM_NCDESTROY
	permissionMenus.Unlock()
	if message == 0x82 {
		permissionComctl32.NewProc("RemoveWindowSubclass").Call(hwnd, permissionSubclass, filePermissionMenuID)
	}
	if message == 0x112 && wparam&0xfff0 == filePermissionMenuID && item != nil { // WM_SYSCOMMAND
		item.view.Dispatch(func() {
			if item.alive() && !item.busy {
				item.busy = true
				item.request(false)
			}
		})
		return 0
	}
	result, _, _ := permissionComctl32.NewProc("DefSubclassProc").Call(hwnd, message, wparam, lparam)
	return result
}

func (m *filePermissionMenu) alive() bool {
	permissionMenus.Lock()
	defer permissionMenus.Unlock()
	return permissionMenus.items[m.hwnd] == m
}

func (m *filePermissionMenu) message(text string, flags uintptr) uintptr {
	caption, _ := windows.UTF16PtrFromString("File access")
	body, _ := windows.UTF16PtrFromString(text)
	result, _, _ := permissionUser32.NewProc("MessageBoxW").Call(m.hwnd, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), flags)
	return result
}

func (m *filePermissionMenu) request(reset bool) {
	finish := func(state uint32, err error) {
		// Never enter a modal message loop inside a WebView2 COM callback.
		m.view.Dispatch(func() {
			if !m.alive() {
				return
			}
			if err != nil {
				m.message("File access settings could not be read or updated. Your documents and draft were not changed.", 0x10)
				m.busy = false
				return
			}
			if !reset && state == 2 {
				if m.message("File reading and writing is blocked for this app in the current profile.\n\nReset this one permission to its default? This does not grant file access or change documents, drafts, or other permissions.", 0x134) == 6 {
					m.request(true)
					return
				}
			} else if reset {
				m.message("The stored file access block is no longer present. No files were written and no automatic access was granted.", 0x40)
			} else {
				m.message("No stored file access block was found for this app. No settings were changed.", 0x40)
			}
			m.busy = false
		})
	}
	if err := webview.FilePermission(m.view, m.origin, reset, finish); err != nil {
		finish(0, err)
	}
}
