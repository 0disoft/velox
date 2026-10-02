//go:build windows

package fileopen

import (
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
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

func TestNativeSaveDialogConfiguration(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := dialogOle32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		t.Fatal("STA initialization failed")
	}
	defer dialogOle32.NewProc("CoUninitialize").Call()
	dialog, err := newSaveDialog("notes.md")
	if err != nil {
		t.Fatal(err)
	}
	defer dialog.call(2)
	var options uint32
	if int32(dialog.call(10, uintptr(unsafe.Pointer(&options)))) < 0 {
		t.Fatal("GetOptions failed")
	}
	if options&0x0211884a != 0x0211884a || options&(0x200|0x1000) != 0 {
		t.Fatalf("unexpected options: %x", options)
	}
	var index uint32
	if int32(dialog.call(6, uintptr(unsafe.Pointer(&index)))) < 0 || index != 2 {
		t.Fatalf("Markdown filter not selected: %d", index)
	}
}

func TestSaveFileTypeDefaults(t *testing.T) {
	for _, tc := range []struct {
		name      string
		index     uint32
		extension string
	}{
		{"Untitled.txt", 1, "txt"}, {"notes", 1, "txt"}, {"notes.text", 1, "text"},
		{"notes.MD", 2, "md"}, {"notes.markdown", 2, "markdown"}, {"notes.log", 3, ""},
	} {
		index, extension := saveFileType(tc.name)
		if index != tc.index || extension != tc.extension {
			t.Fatal(tc.name, index, extension)
		}
	}
}

func TestNativeSaveDialogAllFilesDefault(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := dialogOle32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		t.Fatal("STA initialization failed")
	}
	defer dialogOle32.NewProc("CoUninitialize").Call()
	dialog, err := newSaveDialog("notes.log")
	if err != nil {
		t.Fatal(err)
	}
	defer dialog.call(2)
	var index uint32
	if int32(dialog.call(6, uintptr(unsafe.Pointer(&index)))) < 0 || index != 3 {
		t.Fatalf("all-files filter not selected: %d", index)
	}
}

func TestSaveFileTypeRegistration(t *testing.T) {
	var table [29]uintptr
	types, index, extension := false, uint32(0), "unset"
	table[4] = windows.NewCallback(func(_ *dialogObject, count uintptr, pointer *dialogFilterSpec) uintptr {
		if count != 3 {
			t.Fatal("wrong filter count")
		}
		filters := unsafe.Slice(pointer, int(count))
		for i, expected := range []string{"*.txt;*.text", "*.md;*.markdown", "*.*"} {
			if windows.UTF16PtrToString(filters[i].pattern) != expected || windows.UTF16PtrToString(filters[i].name) == "" {
				t.Fatal("missing filter")
			}
		}
		types = true
		return 0
	})
	table[5] = windows.NewCallback(func(_ uintptr, value uintptr) uintptr { index = uint32(value); return 0 })
	table[22] = windows.NewCallback(func(_ *dialogObject, value *uint16) uintptr {
		extension = windows.UTF16PtrToString(value)
		return 0
	})
	dialog := &dialogObject{table: &table}
	if err := configureSaveFileTypes(dialog, "Untitled.txt"); err != nil || !types || index != 1 || extension != "txt" {
		t.Fatal(types, index, extension, err)
	}
}
