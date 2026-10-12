//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A test executable, never a production switch. Both arms acquire the same
// additional apartment reference; only the balanced arm releases that reference.
func TestHostApartmentProbeChild(t *testing.T) {
	config := os.Getenv("VELOX_APARTMENT_PROBE_CONFIG")
	if config == "" {
		t.Skip("private child of apartment shutdown diagnostic")
	}
	balanced := os.Getenv("VELOX_APARTMENT_PROBE_BALANCED") == "1"
	if os.Getenv("VELOX_APARTMENT_PROBE_FRESH_THREAD") == "1" {
		done := make(chan struct{})
		go func() {
			defer close(done)
			// Exiting with this lock retires the private thread even in the
			// deliberately unbalanced arm; it must not return to Go's pool.
			runtime.LockOSThread()
			runHostApartmentProbe(t, config, balanced, true)
		}()
		<-done
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	runHostApartmentProbe(t, config, balanced, false)
}

func runHostApartmentProbe(t *testing.T, config string, balanced, fresh bool) {
	t.Helper()
	ole := windows.NewLazySystemDLL("ole32.dll")
	initialize := ole.NewProc("CoInitializeEx")
	before, _, _ := queryProbeApartment(ole)
	hr, _, _ := initialize.Call(0, 2)
	if uint32(hr)&0x80000000 != 0 {
		t.Fatalf("probe apartment initialization HRESULT=0x%08x", hr)
	}
	if fresh && (before != "0x800401f0" || uint32(hr) != 0) {
		ole.NewProc("CoUninitialize").Call()
		t.Fatalf("private thread is not a fresh STA: before=%s initialize=0x%08x", before, hr)
	}
	code := runWithStartupNotice([]string{"--config", config}, func(startupFailureKind) {
		t.Error("unexpected startup notice in headless apartment diagnostic")
	})
	if balanced {
		ole.NewProc("CoUninitialize").Call()
	}
	after, afterType, afterQualifier := queryProbeApartment(ole)
	result := struct {
		Balanced       bool   `json:"balanced"`
		Fresh          bool   `json:"freshThread"`
		Before         string `json:"beforeHresult"`
		Init           string `json:"initializeHresult"`
		After          string `json:"afterHresult"`
		AfterType      uint32 `json:"afterApartmentType"`
		AfterQualifier uint32 `json:"afterApartmentQualifier"`
		ExitCode       int    `json:"hostExitCode"`
	}{balanced, fresh, before, fmt.Sprintf("0x%08x", hr), after, afterType, afterQualifier, code}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("VELOX_APARTMENT_PROBE=" + string(encoded))
	if code != 0 {
		t.Fatalf("apartment probe host exit=%d", code)
	}
}

func queryProbeApartment(ole *windows.LazyDLL) (string, uint32, uint32) {
	var apartment, qualifier uint32
	hr, _, _ := ole.NewProc("CoGetApartmentType").Call(
		uintptr(unsafe.Pointer(&apartment)), uintptr(unsafe.Pointer(&qualifier)))
	return fmt.Sprintf("0x%08x", hr), apartment, qualifier
}
