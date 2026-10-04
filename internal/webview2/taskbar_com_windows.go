//go:build windows

package webview2

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var taskbarOle32 = windows.NewLazySystemDLL("ole32.dll")
var taskbarCreate = taskbarOle32.NewProc("CoCreateInstance")
var taskbarClass = windows.GUID{Data1: 0x56fdf344, Data2: 0xfd6d, Data3: 0x11d0, Data4: [8]byte{0x95, 0x8a, 0x00, 0x60, 0x97, 0xc9, 0xa0, 0x90}}
var taskbarInterface = windows.GUID{Data1: 0xea1afb91, Data2: 0x9e28, Data3: 0x4b86, Data4: [8]byte{0x90, 0xe9, 0x9e, 0x9f, 0x8a, 0x5e, 0xef, 0xaf}}

// IUnknown, ITaskbarList, ITaskbarList2, then the first two ITaskbarList3 slots.
type taskbarVTable struct {
	QueryInterface, AddRef, Release                      uintptr
	HrInit, AddTab, DeleteTab, ActivateTab, SetActiveAlt uintptr
	MarkFullscreenWindow                                 uintptr
	SetProgressValue, SetProgressState                   uintptr
}

type taskbarCOM struct{ vtable *taskbarVTable }

func taskbarHRESULT(operation string, result uintptr) error {
	if int32(result) < 0 {
		return fmt.Errorf("%s: HRESULT %#x", operation, uint32(result))
	}
	return nil
}

func createTaskbarClient() (progressClient, error) {
	var client *taskbarCOM
	hr, _, _ := taskbarCreate.Call(uintptr(unsafe.Pointer(&taskbarClass)), 0, 1,
		uintptr(unsafe.Pointer(&taskbarInterface)), uintptr(unsafe.Pointer(&client)))
	if err := taskbarHRESULT("create taskbar interface", hr); err != nil {
		return nil, err
	}
	if client == nil || client.vtable == nil {
		return nil, fmt.Errorf("missing taskbar interface")
	}
	hr, _, _ = syscall.SyscallN(client.vtable.HrInit, uintptr(unsafe.Pointer(client)))
	if err := taskbarHRESULT("initialize taskbar interface", hr); err != nil {
		client.close()
		return nil, err
	}
	return client, nil
}

func (client *taskbarCOM) state(hwnd, flag uintptr) error {
	hr, _, _ := syscall.SyscallN(client.vtable.SetProgressState, uintptr(unsafe.Pointer(client)), hwnd, flag)
	runtime.KeepAlive(client)
	return taskbarHRESULT("set taskbar progress state", hr)
}

func (client *taskbarCOM) value(hwnd uintptr, value uint32) error {
	// Supported hosts are x64: ULONGLONG arguments occupy one 64-bit slot each.
	hr, _, _ := syscall.SyscallN(client.vtable.SetProgressValue, uintptr(unsafe.Pointer(client)), hwnd, uintptr(value), 100)
	runtime.KeepAlive(client)
	return taskbarHRESULT("set taskbar progress value", hr)
}

func (client *taskbarCOM) close() {
	syscall.SyscallN(client.vtable.Release, uintptr(unsafe.Pointer(client)))
	runtime.KeepAlive(client)
}
