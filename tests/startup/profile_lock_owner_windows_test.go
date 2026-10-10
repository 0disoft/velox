package startup_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

type profileLockOwner struct {
	PID       uint32 `json:"pid"`
	StartedAt uint64 `json:"processStartFiletime"`
	Name      string `json:"name"`
}

// RM_PROCESS_INFO has four-byte packing, including RM_UNIQUE_PROCESS/FILETIME.
type restartManagerProcess struct {
	PID         uint32
	Start       windows.Filetime
	Name        [256]uint16
	ServiceName [64]uint16
	Type        uint32
	Status      uint32
	SessionID   uint32
	Restartable int32
}

// Only resource registration and enumeration are used; no shutdown/restart API.
func queryProfileLockOwners(path string) ([]profileLockOwner, error) {
	dll := windows.NewLazySystemDLL("rstrtmgr.dll")
	start, end := dll.NewProc("RmStartSession"), dll.NewProc("RmEndSession")
	register, list := dll.NewProc("RmRegisterResources"), dll.NewProc("RmGetList")
	var session uint32
	var key [33]uint16
	status, _, _ := start.Call(uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&key[0])))
	if status != 0 {
		return nil, fmt.Errorf("RmStartSession: %w", windows.Errno(status))
	}
	defer end.Call(uintptr(session))
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	status, _, _ = register.Call(uintptr(session), 1, uintptr(unsafe.Pointer(&file)), 0, 0, 0, 0)
	runtime.KeepAlive(file)
	if status != 0 {
		return nil, fmt.Errorf("RmRegisterResources: %w", windows.Errno(status))
	}
	var needed, count, reasons uint32
	status, _, _ = list.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&reasons)))
	for attempt := 0; status == uintptr(windows.ERROR_MORE_DATA) && attempt < 3; attempt++ {
		if needed == 0 || needed > 64 {
			return nil, fmt.Errorf("unexpected lock owner count: %d", needed)
		}
		processes := make([]restartManagerProcess, needed)
		count = needed
		status, _, _ = list.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&processes[0])), uintptr(unsafe.Pointer(&reasons)))
		if status == 0 {
			if count > uint32(len(processes)) {
				return nil, fmt.Errorf("invalid returned lock owner count: %d", count)
			}
			owners := make([]profileLockOwner, 0, count)
			for _, process := range processes[:count] {
				owners = append(owners, profileLockOwner{
					PID: process.PID, Name: windows.UTF16ToString(process.Name[:]),
					StartedAt: uint64(process.Start.HighDateTime)<<32 | uint64(process.Start.LowDateTime),
				})
			}
			return owners, nil
		}
	}
	if status != 0 {
		return nil, fmt.Errorf("RmGetList: %w", windows.Errno(status))
	}
	return nil, nil
}

func TestProfileLockOwnerQuery(t *testing.T) {
	if got := unsafe.Sizeof(restartManagerProcess{}); got != 668 {
		t.Fatalf("RM_PROCESS_INFO size = %d, want 668", got)
	}
	path := filepath.Join(t.TempDir(), "owned-lockfile")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	owners, err := queryProfileLockOwners(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range owners {
		if owner.PID == uint32(os.Getpid()) && owner.StartedAt != 0 {
			return
		}
	}
	// Some Windows environments do not enumerate a handle opened with Go's
	// delete-sharing mode; report a failed observation instead of an empty pass.
	t.Fatalf("current process not identified as lockfile user: %+v", owners)
}
