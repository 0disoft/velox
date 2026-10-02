//go:build windows

package fileopen

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var dialogOle32 = windows.NewLazySystemDLL("ole32.dll")
var dialogUser32 = windows.NewLazySystemDLL("user32.dll")
var fileOpenClass = windows.GUID{Data1: 0xdc1c5a9c, Data2: 0xe88a, Data3: 0x4dde, Data4: [8]byte{0xa5, 0xa1, 0x60, 0xf8, 0x2a, 0x20, 0xae, 0xf7}}
var fileOpenInterface = windows.GUID{Data1: 0xd57c7288, Data2: 0xd4ad, Data3: 0x4768, Data4: [8]byte{0xbe, 0x02, 0x9d, 0x96, 0x95, 0x32, 0xd9, 0x60}}
var fileSaveClass = windows.GUID{Data1: 0xc0b4e2f3, Data2: 0xba21, Data3: 0x4773, Data4: [8]byte{0x8d, 0xba, 0x33, 0x5e, 0xc9, 0x46, 0xeb, 0x8b}}
var fileSaveInterface = windows.GUID{Data1: 0x84bccd23, Data2: 0x5fde, Data3: 0x4cdb, Data4: [8]byte{0xae, 0xa4, 0xaf, 0x64, 0xb8, 0x3d, 0x78, 0xab}}

type dialogObject struct{ table *[29]uintptr }

//go:uintptrescapes
func (object *dialogObject) call(index int, args ...uintptr) uint32 {
	values := append([]uintptr{uintptr(unsafe.Pointer(object))}, args...)
	result, _, _ := syscall.SyscallN(object.table[index], values...)
	runtime.KeepAlive(object)
	return uint32(result)
}

func NewWindows(hwnd uintptr, dispatch func(func()), active func() bool, generation func() uint64) *Opener {
	valid := func() bool {
		window, _, _ := dialogUser32.NewProc("IsWindow").Call(hwnd)
		return window != 0 && active()
	}
	return &Opener{Dispatch: dispatch, Active: valid, Generation: generation, Select: func() (string, error) { return selectText(hwnd) }, Read: readSelected}
}

func selectText(hwnd uintptr) (string, error) {
	return selectDialog(hwnd, newTextDialog)
}

func selectDialog(hwnd uintptr, create func() (*dialogObject, error)) (string, error) {
	enabled, _, _ := dialogUser32.NewProc("IsWindowEnabled").Call(hwnd)
	if enabled == 0 {
		return "", ErrBusy
	}
	dialog, err := create()
	if err != nil {
		return "", err
	}
	defer dialog.call(2)
	show := dialog.call(3, hwnd)
	if show == 0x800704c7 {
		return "", nil
	}
	if int32(show) < 0 {
		return "", errors.New("file selection failed")
	}
	var item *dialogObject
	if int32(dialog.call(20, uintptr(unsafe.Pointer(&item)))) < 0 || item == nil {
		return "", ErrUnsupported
	}
	defer item.call(2)
	var path *uint16
	if int32(item.call(5, 0x80058000, uintptr(unsafe.Pointer(&path)))) < 0 || path == nil {
		return "", ErrUnsupported
	}
	defer dialogOle32.NewProc("CoTaskMemFree").Call(uintptr(unsafe.Pointer(path)))
	return windows.UTF16PtrToString(path), nil
}

func newTextDialog() (*dialogObject, error) {
	return createTextDialog(&fileOpenClass, &fileOpenInterface, 0x02001848, "Open text file (read-only, UTF-8, 2 MiB maximum)", "")
}

func NewWindowsSaver(hwnd uintptr, dispatch func(func()), active func() bool, generation func() uint64) *Saver {
	valid := func() bool {
		window, _, _ := dialogUser32.NewProc("IsWindow").Call(hwnd)
		return window != 0 && active()
	}
	return &Saver{Dispatch: dispatch, Active: valid, Generation: generation, Select: func(name string) (string, error) {
		return selectDialog(hwnd, func() (*dialogObject, error) { return newSaveDialog(name) })
	}, Write: writeSelected}
}

func newSaveDialog(name string) (*dialogObject, error) {
	if err := ValidateSaveName(name); err != nil {
		return nil, err
	}
	// Overwrite confirmation, existing parent, no readonly return or test-file creation.
	return createTextDialog(&fileSaveClass, &fileSaveInterface, 0x0211884a, "Save text file (UTF-8, 2 MiB maximum)", name)
}

func createTextDialog(class, iface *windows.GUID, options uint32, caption, suggested string) (*dialogObject, error) {
	var dialog *dialogObject
	hr, _, _ := dialogOle32.NewProc("CoCreateInstance").Call(uintptr(unsafe.Pointer(class)), 0, 1, uintptr(unsafe.Pointer(iface)), uintptr(unsafe.Pointer(&dialog)))
	if int32(hr) < 0 || dialog == nil {
		return nil, errors.New("file selection is unavailable")
	}
	// IFileDialog: single local file, existing path, no cwd change or recent-item write.
	if int32(dialog.call(9, uintptr(options))) < 0 {
		dialog.call(2)
		return nil, ErrUnsupported
	}
	title, _ := windows.UTF16PtrFromString(caption)
	if int32(dialog.call(17, uintptr(unsafe.Pointer(title)))) < 0 {
		dialog.call(2)
		return nil, ErrUnsupported
	}
	runtime.KeepAlive(title)
	if suggested != "" {
		name, _ := windows.UTF16PtrFromString(suggested)
		if int32(dialog.call(15, uintptr(unsafe.Pointer(name)))) < 0 {
			dialog.call(2)
			return nil, ErrUnsupported
		}
		runtime.KeepAlive(name)
	}
	return dialog, nil
}
