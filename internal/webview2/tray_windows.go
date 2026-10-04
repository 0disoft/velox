//go:build windows

package webview2

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"github.com/0disoft/velox/internal/ipc"
	"golang.org/x/sys/windows"
)

const (
	traySubclassID = 0x1f13
	trayMessage    = 0x8000 + 0x113
	trayIconID     = 1
	trayShow       = 1
	trayHide       = 2
	trayQuit       = 3
)

var trayUser32 = windows.NewLazySystemDLL("user32.dll")
var trayShell32 = windows.NewLazySystemDLL("shell32.dll")
var trayInitialize sync.Once
var trayCallback, trayTaskbarMessage uintptr
var trayInitError error
var systemTrays = struct {
	sync.Mutex
	items map[uintptr]*systemTray
}{}

// Matches the full Unicode NOTIFYICONDATA layout, including its version union.
type trayIconData struct {
	Size                uint32
	Window              uintptr
	ID, Flags, Callback uint32
	Icon                uintptr
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Version             uint32
	InfoTitle           [64]uint16
	InfoFlags           uint32
	GUID                windows.GUID
	BalloonIcon         uintptr
}

type systemTray struct {
	data                 trayIconData
	notify               func(uint32, *trayIconData) bool
	isClosing            func() bool
	registered, menuOpen bool
}

func shellNotifyIcon(action uint32, data *trayIconData) bool {
	result, _, _ := trayShell32.NewProc("Shell_NotifyIconW").Call(uintptr(action), uintptr(unsafe.Pointer(data)))
	return result != 0
}

func installSystemTray(hwnd uintptr, title string, enabled bool, closing func() bool) error {
	if !enabled {
		return nil
	}
	_, err := attachSystemTray(hwnd, title, closing, shellNotifyIcon)
	return err
}

func attachSystemTray(hwnd uintptr, title string, closing func() bool, notify func(uint32, *trayIconData) bool) (*systemTray, error) {
	trayInitialize.Do(func() {
		name, _ := windows.UTF16PtrFromString("TaskbarCreated")
		trayTaskbarMessage, _, _ = trayUser32.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(name)))
		if trayTaskbarMessage == 0 {
			trayInitError = fmt.Errorf("register taskbar restart message")
			return
		}
		trayCallback = windows.NewCallback(systemTrayProc)
		systemTrays.items = make(map[uintptr]*systemTray)
	})
	if trayInitError != nil {
		return nil, trayInitError
	}
	icon, _, _ := trayUser32.NewProc("SendMessageW").Call(hwnd, 0x7f, 0, 0) // WM_GETICON, ICON_SMALL
	if icon == 0 {
		icon, _, _ = trayUser32.NewProc("LoadIconW").Call(0, 32512) // Shared default application icon.
	}
	item := &systemTray{notify: notify, isClosing: closing, data: trayIconData{
		Size: uint32(unsafe.Sizeof(trayIconData{})), Window: hwnd, ID: trayIconID,
		Flags: 1 | 2 | 4 | 0x80, Callback: trayMessage, Icon: icon, Version: 4,
	}}
	copy(item.data.Tip[:], trayTooltip(title))
	copy(item.data.InfoTitle[:], trayText(title, 63))
	systemTrays.Lock()
	systemTrays.items[hwnd] = item
	systemTrays.Unlock()
	installed, _, _ := permissionComctl32.NewProc("SetWindowSubclass").Call(hwnd, trayCallback, traySubclassID, 0)
	if installed == 0 || !item.add() {
		permissionComctl32.NewProc("RemoveWindowSubclass").Call(hwnd, trayCallback, traySubclassID)
		systemTrays.Lock()
		delete(systemTrays.items, hwnd)
		systemTrays.Unlock()
		return nil, fmt.Errorf("install system tray")
	}
	return item, nil
}

func trayTooltip(title string) []uint16 {
	return trayText(title, 127)
}

func trayText(title string, limit int) []uint16 {
	text := utf16.Encode([]rune(strings.ReplaceAll(title, "\x00", " ")))
	if len(text) > limit {
		text = text[:limit]
		if text[limit-1] >= 0xd800 && text[limit-1] <= 0xdbff {
			text = text[:limit-1]
		}
	}
	return text
}

func (w nativeWindow) ShowNotification(kind, message string) error {
	hwnd, err := w.handle()
	if err != nil {
		return err
	}
	return showTrayNotification(hwnd, kind, message)
}

func showTrayNotification(hwnd uintptr, kind, message string) error {
	systemTrays.Lock()
	item := systemTrays.items[hwnd]
	systemTrays.Unlock()
	if item == nil {
		return fmt.Errorf("notification requires an enabled tray")
	}
	return item.showNotification(kind, message)
}

func (t *systemTray) showNotification(kind, message string) error {
	if !ipc.ValidNotification(kind, message) || !t.registered || !t.available() {
		return fmt.Errorf("notification is unavailable")
	}
	// Keep balloon fields transient so Explorer recovery never replays old text.
	data := t.data
	data.Flags = 0x10 | 0x40 // NIF_INFO | NIF_REALTIME: no delayed shell queue.
	copy(data.Info[:], utf16.Encode([]rune(message)))
	data.InfoFlags = 0x10 | 0x80 // NIIF_NOSOUND | NIIF_RESPECT_QUIET_TIME.
	switch kind {
	case "info":
		data.InfoFlags |= 1
	case "warning":
		data.InfoFlags |= 2
	case "error":
		data.InfoFlags |= 3
	}
	if !t.notify(1, &data) { // NIM_MODIFY; success means accepted, not displayed.
		return fmt.Errorf("submit tray notification")
	}
	return nil
}

func (t *systemTray) add() bool {
	if !t.notify(0, &t.data) { // NIM_ADD
		return false
	}
	t.registered = true
	if !t.notify(4, &t.data) { // NIM_SETVERSION
		t.remove()
		return false
	}
	return true
}

func (t *systemTray) remove() {
	if t.registered {
		t.notify(2, &t.data) // NIM_DELETE; the window's shared icon remains Windows-owned.
		t.registered = false
	}
}

func (t *systemTray) available() bool {
	systemTrays.Lock()
	alive := systemTrays.items[t.data.Window] == t
	systemTrays.Unlock()
	enabled, _, _ := trayUser32.NewProc("IsWindowEnabled").Call(t.data.Window)
	return alive && enabled != 0 && (t.isClosing == nil || !t.isClosing())
}

func (t *systemTray) show() {
	trayUser32.NewProc("ShowWindow").Call(t.data.Window, 8) // SW_SHOWNA preserves maximized placement.
	minimized, _, _ := trayUser32.NewProc("IsIconic").Call(t.data.Window)
	if minimized != 0 {
		trayUser32.NewProc("ShowWindow").Call(t.data.Window, 9) // SW_RESTORE
	}
	trayUser32.NewProc("SetForegroundWindow").Call(t.data.Window)
}

func (t *systemTray) command(command uintptr) {
	if !t.available() {
		return
	}
	switch command {
	case trayShow:
		t.show()
	case trayHide:
		if t.registered {
			trayUser32.NewProc("ShowWindow").Call(t.data.Window, 0)
		}
	case trayQuit:
		// Reveal the owner first so beforeunload confirmation cannot be stranded.
		t.show()
		trayUser32.NewProc("PostMessageW").Call(t.data.Window, 0x10, 0, 0) // Normal WM_CLOSE, never forced destruction.
	}
}

func (t *systemTray) popup(wparam uintptr) {
	if !t.available() || t.menuOpen {
		return
	}
	t.menuOpen = true
	defer func() { t.menuOpen = false }()
	menu, _, _ := trayUser32.NewProc("CreatePopupMenu").Call()
	if menu == 0 {
		t.show()
		return
	}
	defer trayUser32.NewProc("DestroyMenu").Call(menu)
	for _, entry := range []struct {
		id    uintptr
		label string
	}{{trayShow, "&Open window"}, {trayHide, "&Hide window"}, {trayQuit, "&Quit"}} {
		label, _ := windows.UTF16PtrFromString(entry.label)
		if result, _, _ := trayUser32.NewProc("AppendMenuW").Call(menu, 0, entry.id, uintptr(unsafe.Pointer(label))); result == 0 {
			t.show()
			return
		}
	}
	x, y := int32(int16(wparam&0xffff)), int32(int16((wparam>>16)&0xffff))
	if x == -1 && y == -1 {
		var point struct{ X, Y int32 }
		trayUser32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&point)))
		x, y = point.X, point.Y
	}
	trayUser32.NewProc("SetForegroundWindow").Call(t.data.Window)
	command, _, _ := trayUser32.NewProc("TrackPopupMenu").Call(menu, 0x182, uintptr(x), uintptr(y), 0, t.data.Window, 0)
	// Required for subsequent notification-area popup dismissal.
	trayUser32.NewProc("PostMessageW").Call(t.data.Window, 0, 0, 0)
	t.command(command)
	if t.available() && t.registered {
		t.notify(3, &t.data) // NIM_SETFOCUS after leaving the menu.
	}
}

func systemTrayProc(hwnd, message, wparam, lparam, id, reference uintptr) uintptr {
	systemTrays.Lock()
	item := systemTrays.items[hwnd]
	if message == 0x82 { // WM_NCDESTROY
		delete(systemTrays.items, hwnd)
	}
	systemTrays.Unlock()
	if item != nil {
		switch message {
		case trayMessage:
			if (lparam>>16)&0xffff == trayIconID && item.available() {
				switch lparam & 0xffff {
				case 0x400, 0x401, 0x405: // NIN_SELECT, NIN_KEYSELECT, NIN_BALLOONUSERCLICK.
					item.command(trayShow)
				case 0x7b: // WM_CONTEXTMENU
					item.popup(wparam)
				}
			}
			return 0
		case trayTaskbarMessage:
			item.registered = false // Explorer lost the old registration.
			if (item.isClosing == nil || !item.isClosing()) && !item.add() {
				item.show()
				fmt.Fprintln(os.Stderr, "velox-host: system tray could not be restored; window is visible")
			}
			return 0
		case 0x2: // WM_DESTROY
			item.remove()
		case 0x82:
			item.remove()
			permissionComctl32.NewProc("RemoveWindowSubclass").Call(hwnd, trayCallback, traySubclassID, 0)
		}
	}
	result, _, _ := permissionComctl32.NewProc("DefSubclassProc").Call(hwnd, message, wparam, lparam)
	return result
}
