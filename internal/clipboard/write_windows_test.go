//go:build windows

package clipboard

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNativeClipboardOwnershipAndFailureCleanup(t *testing.T) {
	for _, stage := range []string{"success", "invalid-owner", "alloc", "lock", "open", "empty", "set", "close"} {
		t.Run(stage, func(t *testing.T) {
			var buffer []uint16
			var calls []string
			mark := func(name string) bool { calls = append(calls, name); return stage != name }
			api := nativeAPI{
				valid: func(hwnd uintptr) bool { return mark("invalid-owner") && hwnd == 123 },
				alloc: func(bytes uintptr) uintptr {
					if !mark("alloc") {
						return 0
					}
					buffer = make([]uint16, bytes/2)
					return 7
				},
				lock: func(handle uintptr) uintptr {
					if !mark("lock") {
						return 0
					}
					return 99
				},
				copy: func(destination uintptr, text []uint16) {
					if destination != 99 {
						t.Fatal("invalid memory destination")
					}
					mark("copy")
					copy(buffer, text)
				},
				unlock: func(handle uintptr) { mark("unlock") },
				free:   func(handle uintptr) { mark("free") },
				open:   func(hwnd uintptr) bool { return mark("open") },
				empty:  func() bool { return mark("empty") },
				set:    func(handle uintptr) bool { return handle == 7 && mark("set") },
				close:  func() bool { return mark("close") },
			}
			text := "line\n\ud55c\uae00 \U0001f642"
			err := writeText(123, text, api)
			if stage == "success" {
				if err != nil || !reflect.DeepEqual(calls, []string{"invalid-owner", "alloc", "lock", "copy", "unlock", "open", "empty", "set", "close"}) {
					t.Fatalf("calls=%v err=%v", calls, err)
				}
				want, _ := windows.UTF16FromString(text)
				if !reflect.DeepEqual(buffer, want) {
					t.Fatal("UTF-16 text or terminator changed")
				}
				return
			}
			wantErr := ErrNative
			if stage == "open" {
				wantErr = ErrBusy
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("error=%v", err)
			}
			joined := strings.Join(calls, ",")
			free := strings.Contains(joined, "free")
			if free != (stage != "invalid-owner" && stage != "alloc" && stage != "close") {
				t.Fatalf("allocation ownership: %v", calls)
			}
			closed := strings.Contains(joined, "close")
			if closed != (stage == "empty" || stage == "set" || stage == "close") {
				t.Fatalf("clipboard lock cleanup: %v", calls)
			}
			if (stage == "open" || stage == "alloc" || stage == "lock") && strings.Contains(joined, "empty") {
				t.Fatal("clipboard cleared before preflight completed")
			}
		})
	}
}

func TestNativeClipboardRejectsInvalidTextWithoutOSAccess(t *testing.T) {
	for _, text := range []string{"a\x00b", strings.Repeat("x", MaxTextBytes+1), string([]byte{0xff})} {
		if err := writeText(123, text, nativeAPI{}); err == nil {
			t.Fatal("invalid text accepted")
		}
	}
	if err := writeText(0, "hello", nativeAPI{}); !errors.Is(err, ErrNative) {
		t.Fatal("missing owner accepted")
	}
}

func TestWindowsUnicodeAllocationWithoutClipboardAccess(t *testing.T) {
	api := newNativeAPI()
	want, _ := windows.UTF16FromString("\ud55c\uae00\n\U0001f642")
	handle := api.alloc(uintptr(len(want) * 2))
	if handle == 0 {
		t.Fatal("GlobalAlloc failed")
	}
	defer api.free(handle)
	address := api.lock(handle)
	if address == 0 {
		t.Fatal("GlobalLock failed")
	}
	defer api.unlock(handle)
	api.copy(address, want)
	got := make([]uint16, len(want))
	windows.NewLazySystemDLL("ntdll.dll").NewProc("RtlMoveMemory").Call(
		uintptr(unsafe.Pointer(&got[0])), address, uintptr(len(got)*2))
	if !reflect.DeepEqual(got, want) {
		t.Fatal("Native Unicode allocation readback mismatch")
	}
}
