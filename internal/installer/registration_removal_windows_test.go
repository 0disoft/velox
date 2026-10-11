package installer

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"

	"github.com/0disoft/velox/internal/buildreport"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func registrationRemovalFixture(t *testing.T) (Record, string, string) {
	t.Helper()
	root := t.TempDir()
	registryRoot := `Software\VeloxInstallerTests\removal-` + filepath.Base(root) + `\`
	record := Record{SchemaVersion: stateSchema, Directory: filepath.Join(root, "app"),
		App: buildreport.App{ID: "dev.velox.test", Name: "Test", Version: "1.0.0"}}
	path := registryRoot + record.App.ID
	key, existed, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
	if err != nil || existed {
		t.Fatalf("isolated registration creation: existed=%v err=%v", existed, err)
	}
	if err := writeWindowsRegistration(key, record); err != nil {
		key.Close()
		t.Fatal(err)
	}
	key.Close()
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, path)
		_ = registry.DeleteKey(registry.CURRENT_USER, registryRoot[:len(registryRoot)-1])
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
	})
	return record, registryRoot, filepath.Join(root, "menu")
}

func TestWindowsRegistrationChangedDuringShortcutRemovalIsPreserved(t *testing.T) {
	record, root, _ := registrationRemovalFixture(t)
	path := root + record.App.ID
	err := removeWindowsRegistration(record, path, func() error {
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
		if err != nil {
			return err
		}
		defer key.Close()
		return key.SetStringValue("UserNotes", "late registry data")
	})
	if err == nil {
		t.Fatal("removed registration changed during shortcut IO")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	note, _, err := key.GetStringValue("UserNotes")
	if err != nil || note != "late registry data" {
		t.Fatalf("late registry data changed: %v", err)
	}
}

func TestWindowsRegistrationPathReplacementKeepsForeignKey(t *testing.T) {
	record, root, _ := registrationRemovalFixture(t)
	path := root + record.App.ID
	parent := root[:len(root)-1]
	renamed := record.App.ID + "-moved"
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, root+renamed) })
	rename := windows.NewLazySystemDLL("advapi32.dll").NewProc("RegRenameKey")
	if err := rename.Find(); err != nil {
		t.Fatal(err)
	}
	err := removeWindowsRegistration(record, path, func() error {
		key, err := registry.OpenKey(registry.CURRENT_USER, parent, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		defer key.Close()
		oldName, _ := windows.UTF16PtrFromString(record.App.ID)
		newName, _ := windows.UTF16PtrFromString(renamed)
		status, _, _ := rename.Call(uintptr(key), uintptr(unsafe.Pointer(oldName)), uintptr(unsafe.Pointer(newName)))
		if status != 0 {
			return syscall.Errno(status)
		}
		foreign, exists, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		defer foreign.Close()
		if exists {
			return errors.New("original registry path was not released by rename")
		}
		return foreign.SetStringValue("UserNotes", "foreign replacement")
	})
	if err != nil {
		t.Fatalf("verified key-handle removal failed: %v", err)
	}
	foreign, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("replacement registry key was deleted: %v", err)
	}
	defer foreign.Close()
	note, _, err := foreign.GetStringValue("UserNotes")
	if err != nil || note != "foreign replacement" {
		t.Fatalf("replacement data changed: %v", err)
	}
	old, err := registry.OpenKey(registry.CURRENT_USER, root+renamed, registry.QUERY_VALUE)
	if err == nil {
		old.Close()
		t.Fatal("original verified key was not removed")
	}
	if !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestWindowsRegistrationChangedAfterValidationIsPreserved(t *testing.T) {
	record, root, menu := registrationRemovalFixture(t)
	path := root + record.App.ID
	err := unregisterWindows(record, menu, root, func(record Record) error {
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		defer key.Close()
		if err := validateWindowsRegistration(key, record); err != nil {
			return err
		}
		return key.SetStringValue("VeloxInstaller", "foreign owner")
	})
	if err == nil {
		t.Fatal("removed registration changed after its earlier validation")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("foreign registration disappeared: %v", err)
	}
	defer key.Close()
	owner, _, err := key.GetStringValue("VeloxInstaller")
	if err != nil || owner != "foreign owner" {
		t.Fatalf("foreign registration changed: %v", err)
	}
}
