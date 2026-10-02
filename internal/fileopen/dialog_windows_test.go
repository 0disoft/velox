//go:build windows

package fileopen

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestNativeDialogConfiguration(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := dialogOle32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		t.Fatal("STA initialization failed")
	}
	defer dialogOle32.NewProc("CoUninitialize").Call()
	dialog, err := newTextDialog()
	if err != nil {
		t.Fatal(err)
	}
	defer dialog.call(2)
	var options uint32
	if int32(dialog.call(10, uintptr(unsafe.Pointer(&options)))) < 0 {
		t.Fatal("GetOptions failed")
	}
	if options&0x02001848 != 0x02001848 || options&0x200 != 0 {
		t.Fatalf("unexpected dialog options: %x", options)
	}
}
