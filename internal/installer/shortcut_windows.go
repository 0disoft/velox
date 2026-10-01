package installer

import (
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type comObject struct{ vtable *[32]uintptr }

func (object *comObject) call(slot int, args ...uintptr) error {
	arguments := append([]uintptr{uintptr(unsafe.Pointer(object))}, args...)
	result, _, _ := syscall.SyscallN(object.vtable[slot], arguments...)
	if int32(result) < 0 {
		return fmt.Errorf("Shell link method failed: HRESULT 0x%08x", uint32(result))
	}
	return nil
}

func (object *comObject) release() {
	syscall.SyscallN(object.vtable[2], uintptr(unsafe.Pointer(object)))
}

func createShortcut(path, target, description string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	hr, _, _ := ole.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		return fmt.Errorf("initialize Shell link COM: HRESULT 0x%08x", uint32(hr))
	}
	defer ole.NewProc("CoUninitialize").Call()
	clsid := windows.GUID{Data1: 0x00021401, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	iid := windows.GUID{Data1: 0x000214f9, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	var link *comObject
	hr, _, _ = ole.NewProc("CoCreateInstance").Call(uintptr(unsafe.Pointer(&clsid)), 0, 1, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&link)))
	if int32(hr) < 0 {
		return fmt.Errorf("create Shell link: HRESULT 0x%08x", uint32(hr))
	}
	defer link.release()
	for _, field := range []struct {
		slot int
		text string
	}{{20, target}, {9, filepath.Dir(target)}, {7, description}} {
		value, err := windows.UTF16PtrFromString(field.text)
		if err != nil {
			return err
		}
		if err := link.call(field.slot, uintptr(unsafe.Pointer(value))); err != nil {
			return err
		}
		runtime.KeepAlive(value)
	}
	persistID := windows.GUID{Data1: 0x0000010b, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	var persist *comObject
	if err := link.call(0, uintptr(unsafe.Pointer(&persistID)), uintptr(unsafe.Pointer(&persist))); err != nil {
		return err
	}
	defer persist.release()
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	err = persist.call(6, uintptr(unsafe.Pointer(name)), 1)
	runtime.KeepAlive(name)
	return err
}
