//go:build windows

package main

import (
	"fmt"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Test-only comparison on the private STA after the real host loop returns.
// Neither arm changes production teardown or waits on a browser/process handle.
func runPostLoopProbe(mode string) (int, float64, error) {
	if mode == "" {
		return 0, 0, nil
	}
	if mode != "idle" && mode != "pump" {
		return 0, 0, fmt.Errorf("invalid post-loop probe mode")
	}
	started := time.Now()
	if mode == "idle" {
		time.Sleep(time.Second)
		return 0, float64(time.Since(started)) / float64(time.Millisecond), nil
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	peek := user32.NewProc("PeekMessageW")
	translate := user32.NewProc("TranslateMessage")
	dispatch := user32.NewProc("DispatchMessageW")
	wait := user32.NewProc("MsgWaitForMultipleObjectsEx")
	var msg struct {
		Hwnd           windows.Handle
		Message        uint32
		WParam, LParam uintptr
		Time           uint32
		X, Y           int32
		Private        uint32
	}
	deadline := started.Add(time.Second)
	messages := 0
	for time.Now().Before(deadline) {
		r, _, _ := peek.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
		if r != 0 {
			if msg.Message != 0x0012 { // WM_QUIT was already consumed by Run.
				translate.Call(uintptr(unsafe.Pointer(&msg)))
				dispatch.Call(uintptr(unsafe.Pointer(&msg)))
				messages++
			}
			continue
		}
		remaining := time.Until(deadline).Milliseconds()
		if remaining <= 0 {
			break
		}
		wait.Call(0, 0, uintptr(remaining), 0x04ff, 0x0004)
	}
	return messages, float64(time.Since(started)) / float64(time.Millisecond), nil
}

func TestPostLoopProbeRejectsUnknownModes(t *testing.T) {
	for _, mode := range []string{"", "unknown"} {
		messages, elapsed, err := runPostLoopProbe(mode)
		if messages != 0 || elapsed != 0 || (err != nil) != (mode != "") {
			t.Fatalf("mode=%q messages=%d elapsed=%f error=%v", mode, messages, elapsed, err)
		}
	}
}
