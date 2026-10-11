package installer

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var deleteRegistryHandle = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtDeleteKey")

func removeWindowsRegistration(record Record, path string, removeShortcut func() error) error {
	if err := deleteRegistryHandle.Find(); err != nil {
		return fmt.Errorf("preserving registration; handle removal unavailable: %w", err)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ|windows.DELETE)
	if errors.Is(err, registry.ErrNotExist) {
		return removeShortcut()
	}
	if err != nil {
		return err
	}
	defer key.Close()
	if err := validateWindowsRegistration(key, record); err != nil {
		return err
	}
	if err := removeShortcut(); err != nil {
		return err
	}
	// Shortcut IO may take time. Revalidate the still-open key, then remove that
	// object rather than a newly created registration at the same path.
	if err := validateWindowsRegistration(key, record); err != nil {
		return err
	}
	status, _, _ := deleteRegistryHandle.Call(uintptr(key))
	if status != 0 {
		return fmt.Errorf("preserving registration; remove verified key handle: %w", windows.NTStatus(status))
	}
	return nil
}
