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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	initialize := ole.NewProc("CoInitializeEx")
	before := queryProbeApartment(ole)
	hr, _, _ := initialize.Call(0, 2)
	if uint32(hr)&0x80000000 != 0 {
		t.Fatalf("probe apartment initialization HRESULT=0x%08x", hr)
	}
	code := runWithStartupNotice([]string{"--config", config}, func(startupFailureKind) {
		t.Error("unexpected startup notice in headless apartment diagnostic")
	})
	if balanced {
		ole.NewProc("CoUninitialize").Call()
	}
	result := struct {
		Balanced bool   `json:"balanced"`
		Before   string `json:"beforeHresult"`
		Init     string `json:"initializeHresult"`
		After    string `json:"afterHresult"`
		ExitCode int    `json:"hostExitCode"`
	}{balanced, before, fmt.Sprintf("0x%08x", hr), queryProbeApartment(ole), code}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("VELOX_APARTMENT_PROBE=" + string(encoded))
	if code != 0 {
		t.Fatalf("apartment probe host exit=%d", code)
	}
}

func queryProbeApartment(ole *windows.LazyDLL) string {
	var apartment, qualifier uint32
	hr, _, _ := ole.NewProc("CoGetApartmentType").Call(
		uintptr(unsafe.Pointer(&apartment)), uintptr(unsafe.Pointer(&qualifier)))
	return fmt.Sprintf("0x%08x", hr)
}
