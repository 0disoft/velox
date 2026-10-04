//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/0disoft/velox/internal/platformversion"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const themeSubclassID = 0x1f18

var themeDWM = windows.NewLazySystemDLL("dwmapi.dll")
var themeSetAttribute = themeDWM.NewProc("DwmSetWindowAttribute")
var themeSystemParameters = user32Window.NewProc("SystemParametersInfoW")
var themeOnce sync.Once
var themeCallback uintptr
var themeOwners = struct {
	sync.Mutex
	items map[uintptr]*windowThemeOwner
}{items: make(map[uintptr]*windowThemeOwner)}

type themePlatform struct {
	supported func() bool
	read      func() (bool, error)
	apply     func(uintptr, bool) error
}

type windowThemeOwner struct {
	platform              themePlatform
	known, dark, updating bool
}

type themeHighContrast struct {
	Size, Flags uint32
	Scheme      uintptr
}

func nativeThemeSupported() bool {
	version := platformversion.Current()
	// Attribute 20 is documented starting with Windows 11 build 22000.
	return version.Major >= 10 && version.Build >= 22000 &&
		themeSetAttribute.Find() == nil && themeSystemParameters.Find() == nil
}

func readSystemDarkTheme() (bool, error) {
	contrast := themeHighContrast{Size: uint32(unsafe.Sizeof(themeHighContrast{}))}
	if ok, _, err := themeSystemParameters.Call(0x42, uintptr(contrast.Size), uintptr(unsafe.Pointer(&contrast)), 0); ok == 0 {
		return false, fmt.Errorf("read high contrast: %v", err)
	}
	if contrast.Flags&1 != 0 {
		return false, nil
	} // HCF_HIGHCONTRASTON takes priority.
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer key.Close()
	light, kind, err := key.GetIntegerValue("AppsUseLightTheme")
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return kind == registry.DWORD && appThemeIsDark(false, light), nil
}

func applySystemDarkTheme(hwnd uintptr, dark bool) error {
	var enabled uint32
	if dark {
		enabled = 1
	}
	result, _, _ := themeSetAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&enabled)), unsafe.Sizeof(enabled))
	if int32(result) < 0 {
		return fmt.Errorf("apply system frame theme: HRESULT %#x", uint32(result))
	}
	return nil
}

func installSystemTheme(hwnd uintptr, enabled bool) error {
	return installWindowTheme(hwnd, enabled, themePlatform{
		supported: nativeThemeSupported, read: readSystemDarkTheme, apply: applySystemDarkTheme,
	})
}

func installWindowTheme(hwnd uintptr, enabled bool, platform themePlatform) error {
	if !enabled || !platform.supported() {
		return nil
	}
	if valid, _, _ := isNativeWindow.Call(hwnd); valid == 0 {
		return errors.New("theme window is unavailable")
	}
	themeOnce.Do(func() { themeCallback = windows.NewCallback(windowThemeProc) })
	item := &windowThemeOwner{platform: platform}
	themeOwners.Lock()
	themeOwners.items[hwnd] = item
	themeOwners.Unlock()
	if ok, _, err := stateSetSubclass.Call(hwnd, themeCallback, themeSubclassID, 0); ok == 0 {
		themeOwners.Lock()
		delete(themeOwners.items, hwnd)
		themeOwners.Unlock()
		return fmt.Errorf("install window theme handler: %v", err)
	}
	if err := item.refresh(hwnd, false); err != nil {
		fmt.Fprintln(os.Stderr, "velox-host: system frame theme unavailable; existing appearance retained")
	}
	return nil
}

func (item *windowThemeOwner) refresh(hwnd uintptr, force bool) error {
	// DWM may synchronously send theme messages back to this same UI thread.
	if item.updating {
		return nil
	}
	item.updating = true
	defer func() { item.updating = false }()
	dark, err := item.platform.read()
	if err != nil {
		return err
	}
	if !force && item.known && item.dark == dark {
		return nil
	}
	if err := item.platform.apply(hwnd, dark); err != nil {
		return err
	}
	item.known, item.dark = true, dark
	return nil
}

func windowThemeProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	themeOwners.Lock()
	item := themeOwners.items[hwnd]
	if message == 0x82 {
		delete(themeOwners.items, hwnd)
	}
	themeOwners.Unlock()
	if message == 0x82 {
		stateRemoveSubclass.Call(hwnd, themeCallback, themeSubclassID)
	}
	result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
	if item != nil {
		switch message {
		case 0x1a, 0x15, 0x31a: // WM_SETTINGCHANGE, WM_SYSCOLORCHANGE, WM_THEMECHANGED
			if err := item.refresh(hwnd, message == 0x31a); err != nil {
				fmt.Fprintln(os.Stderr, "velox-host: system frame theme update skipped; existing appearance retained")
			}
		}
	}
	return result
}
