//go:build windows

package clipboard

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestNativeClipboardReadBoundsAndCleanup(t *testing.T) {
	for _, stage := range []string{"success", "empty-text", "boundary", "padding", "owner", "open", "unsupported", "get", "size", "lock", "close", "unterminated", "surrogate", "low-surrogate", "utf8-budget", "huge-allocation"} {
		t.Run(stage, func(t *testing.T) {
			buffer := append(utf16.Encode([]rune("\ud55c\uae00\n\U0001f642")), 0)
			switch stage {
			case "empty-text":
				buffer = []uint16{0}
			case "boundary":
				buffer = append(utf16.Encode([]rune(strings.Repeat("x", MaxTextBytes))), 0)
			case "padding":
				buffer = append(buffer, 0xffff, 0xd800)
			case "unterminated":
				buffer = []uint16{'x'}
			case "surrogate":
				buffer = []uint16{0xd800, 0}
			case "low-surrogate":
				buffer = []uint16{0xdc00, 0}
			case "utf8-budget":
				buffer = append(utf16.Encode([]rune(strings.Repeat("\ud55c", MaxTextBytes/3+1))), 0)
			case "huge-allocation":
				buffer = make([]uint16, MaxTextBytes+1)
				for i := range buffer {
					buffer[i] = 'x'
				}
			}
			var calls []string
			mark := func(name string) bool { calls = append(calls, name); return stage != name }
			api := readAPI{
				valid:     func(hwnd uintptr) bool { return mark("owner") && hwnd == 123 },
				open:      func(uintptr) bool { return mark("open") },
				available: func() bool { return mark("unsupported") },
				get: func() uintptr {
					if !mark("get") {
						return 0
					}
					return 7
				},
				size: func(uintptr) uintptr {
					if !mark("size") {
						return 0
					}
					if stage == "huge-allocation" {
						return 1 << 30
					}
					return uintptr(len(buffer) * 2)
				},
				lock: func(uintptr) uintptr {
					if !mark("lock") {
						return 0
					}
					return 99
				},
				copy: func(destination []uint16, address uintptr) {
					mark("copy")
					if address != 99 || len(destination) > MaxTextBytes+1 {
						t.Fatal("unbounded or invalid copy")
					}
					copy(destination, buffer)
				},
				unlock: func(uintptr) { mark("unlock") },
				close:  func() bool { return mark("close") },
			}
			text, err := readText(123, api)
			if stage == "success" || stage == "empty-text" || stage == "boundary" || stage == "padding" {
				want := "\ud55c\uae00\n\U0001f642"
				if stage == "empty-text" {
					want = ""
				}
				if stage == "boundary" {
					want = strings.Repeat("x", MaxTextBytes)
				}
				if text != want || err != nil {
					t.Fatal(text, err)
				}
			} else if text != "" || err == nil {
				t.Fatal("failure leaked text or passed", stage)
			}
			if stage == "success" && !reflect.DeepEqual(calls, []string{"owner", "open", "unsupported", "get", "size", "lock", "copy", "unlock", "close"}) {
				t.Fatal(calls)
			}
			joined := strings.Join(calls, ",")
			if strings.Contains(joined, "close") != (stage != "owner" && stage != "open") {
				t.Fatal("clipboard lock leak", calls)
			}
			copied := strings.Contains(joined, "copy")
			if copied != strings.Contains(joined, "unlock") {
				t.Fatal("global memory lock leak", calls)
			}
			if stage == "open" && !errors.Is(err, ErrBusy) {
				t.Fatal(err)
			}
			if (stage == "utf8-budget" || stage == "huge-allocation") && !errors.Is(err, ErrTooLarge) {
				t.Fatal(err)
			}
		})
	}
}

func TestWindowsReadMemoryCopyWithoutClipboardAccess(t *testing.T) {
	w := newNativeAPI()
	r := newReadAPI()
	want := append(utf16.Encode([]rune("\ud55c\uae00 \U0001f642")), 0)
	handle := w.alloc(uintptr(len(want) * 2))
	if handle == 0 {
		t.Fatal("GlobalAlloc failed")
	}
	defer w.free(handle)
	address := r.lock(handle)
	if address == 0 {
		t.Fatal("GlobalLock failed")
	}
	defer r.unlock(handle)
	if r.size(handle) < uintptr(len(want)*2) {
		t.Fatal("GlobalSize underreported")
	}
	w.copy(address, want)
	got := make([]uint16, len(want))
	r.copy(got, address)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("native memory copy mismatch")
	}
}
