//go:build windows

package webview2

import (
	"errors"
	"testing"
	"unsafe"
)

func themeOwnerFor(hwnd uintptr) *windowThemeOwner {
	themeOwners.Lock()
	defer themeOwners.Unlock()
	return themeOwners.items[hwnd]
}

func TestSystemThemeDefaultAndUnsupportedDoNothing(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	checked, reads, writes := 0, 0, 0
	p := themePlatform{
		supported: func() bool { checked++; return false },
		read:      func() (bool, error) { reads++; return true, nil },
		apply:     func(uintptr, bool) error { writes++; return nil },
	}
	if err := installWindowTheme(hwnd, false, p); err != nil {
		t.Fatal(err)
	}
	if checked != 0 || themeOwnerFor(hwnd) != nil {
		t.Fatal("default performed theme work")
	}
	if err := installWindowTheme(hwnd, true, p); err != nil {
		t.Fatal(err)
	}
	if checked != 1 || reads != 0 || writes != 0 || themeOwnerFor(hwnd) != nil {
		t.Fatal("unsupported performed theme work")
	}
	p.supported = func() bool { return true }
	if err := installWindowTheme(0, true, p); err == nil {
		t.Fatal("invalid HWND accepted")
	}
}

func TestSystemThemeEventsRetryReentryAndCleanup(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	var before, after windowRect
	minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&before)))
	light := uint64(1)
	contrast := false
	reads := 0
	var values []bool
	var readErr, applyErr error
	p := themePlatform{
		supported: func() bool { return true },
		read:      func() (bool, error) { reads++; return appThemeIsDark(contrast, light), readErr },
		apply: func(hwnd uintptr, dark bool) error {
			values = append(values, dark)
			// Exercise synchronous theme-message reentry without recursion.
			stateUser32.NewProc("SendMessageW").Call(hwnd, 0x1a, 0, 0)
			return applyErr
		},
	}
	if err := installWindowTheme(hwnd, true, p); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || len(values) != 1 || values[0] {
		t.Fatal(reads, values)
	}
	send := func(message uintptr) { stateUser32.NewProc("SendMessageW").Call(hwnd, message, 0, 0) }
	send(0)
	if reads != 1 {
		t.Fatal("unrelated event read preferences")
	}
	light = 0
	send(0x1a)
	send(0x1a)
	if len(values) != 2 || !values[1] {
		t.Fatal("dark/dedup failed", values)
	}
	contrast = true
	send(0x15)
	if len(values) != 3 || values[2] {
		t.Fatal("high contrast did not release dark override", values)
	}
	send(0x31a)
	if len(values) != 4 || values[3] {
		t.Fatal("OS theme reset was not reapplied", values)
	}
	contrast = false
	readErr = errors.New("temporary preference failure")
	send(0x1a)
	if len(values) != 4 || themeOwnerFor(hwnd).dark {
		t.Fatal("read failure changed applied state")
	}
	readErr = nil
	applyErr = errors.New("temporary DWM failure")
	send(0x1a)
	if themeOwnerFor(hwnd).dark {
		t.Fatal("failed native application was cached")
	}
	applyErr = nil
	send(0x1a)
	if len(values) != 6 || !themeOwnerFor(hwnd).dark {
		t.Fatal("retry failed", values)
	}
	minimumGetRect.Call(hwnd, uintptr(unsafe.Pointer(&after)))
	if before != after {
		t.Fatal("theme changed geometry")
	}
	if visible, _, _ := stateUser32.NewProc("IsWindowVisible").Call(hwnd); visible != 0 {
		t.Fatal("theme showed hidden window")
	}
	if fg, _, _ := stateUser32.NewProc("GetForegroundWindow").Call(); fg == hwnd {
		t.Fatal("theme activated window")
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if themeOwnerFor(hwnd) != nil {
		t.Fatal("theme owner retained on destruction")
	}
}

func TestSystemThemeNativeAttributeAndABI(t *testing.T) {
	stateTestThread(t)
	if unsafe.Sizeof(themeHighContrast{}) != 16 || unsafe.Offsetof(themeHighContrast{}.Scheme) != 8 {
		t.Fatal("HIGHCONTRAST ABI mismatch")
	}
	if !nativeThemeSupported() {
		t.Skip("documented DWM theme attribute is unavailable")
	}
	hwnd := stateTestWindow(t)
	if err := installSystemTheme(hwnd, true); err != nil {
		t.Fatal(err)
	}
	item := themeOwnerFor(hwnd)
	if item == nil || !item.known {
		t.Fatal("native theme was not applied")
	}
	var got uint32
	result, _, _ := themeDWM.NewProc("DwmGetWindowAttribute").Call(hwnd, 20, uintptr(unsafe.Pointer(&got)), unsafe.Sizeof(got))
	if int32(result) < 0 || (got != 0) != item.dark {
		t.Fatalf("native attribute: %#x, %d", result, got)
	}
}
