package installer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func TestWindowsInstallShortcutRegistrationAndRemoval(t *testing.T) {
	source, setup, _ := fixture(t)
	root := t.TempDir()
	registryRoot := `Software\VeloxInstallerTests\` + filepath.Base(root) + `\`
	env := windowsEnvironment(filepath.Join(root, "apps"), filepath.Join(root, "menu"), registryRoot)
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, registryRoot+"dev.velox.test")
		_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
	})
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, registryRoot+result.AppID, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := key.GetStringValue("UninstallString")
	key.Close()
	if err != nil || !strings.Contains(command, `uninstall.exe" --uninstall dev.velox.test`) {
		t.Fatalf("uninstall command = %q: %v", command, err)
	}
	link := filepath.Join(root, "menu", result.AppID+".lnk")
	if info, err := os.Stat(link); err != nil || info.Size() < 76 {
		t.Fatalf("shortcut = %v %v", info, err)
	}
	if _, err := uninstall(result.AppID, env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(link); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("shortcut remains")
	}
	key, err = registry.OpenKey(registry.CURRENT_USER, registryRoot+result.AppID, registry.QUERY_VALUE)
	if err == nil {
		key.Close()
		t.Fatal("registration remains")
	}
	if !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestWindowsInterruptedAfterShortcutPublication(t *testing.T) {
	source, setup, _ := fixture(t)
	root := t.TempDir()
	registryRoot := `Software\VeloxInstallerTests\` + filepath.Base(root) + `\`
	env := windowsEnvironment(filepath.Join(root, "apps"), filepath.Join(root, "menu"), registryRoot)
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, registryRoot+"dev.velox.test")
		_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
	})
	prepare := env.prepare
	stopped := false
	env.prepare = func(record Record) (registration, error) {
		prepared, err := prepare(record)
		if err != nil {
			return registration{}, err
		}
		publish := prepared.publish
		prepared.publish = func() error {
			if err := publish(); err != nil {
				return err
			}
			stopped = true
			panic("injected stop after shortcut publication")
		}
		return prepared, nil
	}
	func() {
		defer func() { _ = recover() }()
		_, _ = install(source, setup, env)
	}()
	if !stopped {
		t.Fatal("did not reach shortcut publication")
	}
	link := filepath.Join(root, "menu", "dev.velox.test.lnk")
	original, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(link, []byte("user-modified shortcut"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstall("dev.velox.test", env); err == nil {
		t.Fatal("removed a user-modified shortcut after interrupted installation")
	}
	if data, err := os.ReadFile(link); err != nil || string(data) != "user-modified shortcut" {
		t.Fatalf("modified shortcut was not preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.root, "dev.velox.test", "uninstall.exe")); err != nil {
		t.Fatalf("shortcut refusal partially removed the installation: %v", err)
	}
	if err := os.WriteFile(link, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstall("dev.velox.test", env); err != nil {
		t.Fatalf("interrupted installation cannot be removed: %v", err)
	}
	if _, err := os.Stat(link); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shortcut remains: %v", err)
	}
}

func TestWindowsStateReplacementFailureKeepsOldRecord(t *testing.T) {
	source, setup, env := fixture(t)
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(result.Directory, stateFile)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if handle != windows.InvalidHandle {
			_ = windows.CloseHandle(handle)
		}
	}()
	record, err := readState(result.Directory, result.AppID)
	if err != nil {
		t.Fatal(err)
	}
	record.ShortcutHash = strings.Repeat("a", 64)
	if err := writeState(result.Directory, record); err == nil {
		t.Fatal("replacement succeeded while deletion/rename was locked")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("previous ownership record changed: %v", err)
	}
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = windows.InvalidHandle
	if _, err := uninstall(result.AppID, env); err != nil {
		t.Fatalf("old record is no longer usable: %v", err)
	}
}
