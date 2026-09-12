//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestDPIConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		errors    []error
		current   int
		wantError bool
	}{
		{"v2", []error{nil}, 2, false},
		{"server-v1", []error{windows.ERROR_INVALID_PARAMETER, nil}, 2, false},
		{"already-per-monitor", []error{windows.ERROR_ACCESS_DENIED}, 2, false},
		{"incompatible-existing-mode", []error{windows.ERROR_ACCESS_DENIED}, 0, true},
		{"fallback-failed", []error{windows.ERROR_INVALID_PARAMETER, windows.ERROR_ACCESS_DENIED}, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := configureDPI(func(context uintptr) error {
				want := ^uintptr(3)
				if calls == 1 {
					want = ^uintptr(2)
				}
				if context != want || calls >= len(tc.errors) {
					t.Fatal("unexpected DPI context request")
				}
				result := tc.errors[calls]
				calls++
				return result
			}, func() int { return tc.current })
			if (err != nil) != tc.wantError || calls != len(tc.errors) {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestDPIActualProcessMode(t *testing.T) {
	if err := enablePerMonitorDPI(); err != nil {
		t.Fatal(err)
	}
	if err := enablePerMonitorDPI(); err != nil {
		t.Fatalf("repeat configuration: %v", err)
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	context, _, _ := user32.NewProc("GetThreadDpiAwarenessContext").Call()
	mode, _, _ := user32.NewProc("GetAwarenessFromDpiAwarenessContext").Call(context)
	if mode != 2 {
		t.Fatalf("DPI awareness = %d, want per-monitor", mode)
	}
}
