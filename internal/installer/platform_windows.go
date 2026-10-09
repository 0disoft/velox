package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/velox/internal/safefs"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const uninstallRoot = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Velox.`

func userEnvironment() (environment, error) {
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		return environment{}, err
	}
	programs, err := windows.KnownFolderPath(windows.FOLDERID_Programs, 0)
	if err != nil {
		return environment{}, err
	}
	return windowsEnvironment(filepath.Join(local, "Programs", "Velox"), filepath.Join(programs, "Velox"), uninstallRoot), nil
}

func windowsEnvironment(root, shortcuts, registryRoot string) environment {
	shortcut := func(record Record) string { return filepath.Join(shortcuts, record.App.ID+".lnk") }
	validate := func(record Record) error {
		key, err := registry.OpenKey(registry.CURRENT_USER, registryRoot+record.App.ID, registry.QUERY_VALUE)
		if err == nil {
			defer key.Close()
			owner, _, e1 := key.GetStringValue("VeloxInstaller")
			location, _, e2 := key.GetStringValue("InstallLocation")
			if e1 != nil || e2 != nil || owner != stateSchema || !strings.EqualFold(location, record.Directory) {
				return errors.New("uninstall registration belongs to another installation")
			}
		} else if !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		path := shortcut(record)
		if err := safefs.RejectLinkedComponents(path); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if record.ShortcutHash == "" || hex.EncodeToString(digest[:]) != record.ShortcutHash {
			return errors.New("Start Menu shortcut changed; move it before uninstalling")
		}
		return nil
	}
	return environment{
		root: root,
		check: func(record Record) error {
			key, err := registry.OpenKey(registry.CURRENT_USER, registryRoot+record.App.ID, registry.QUERY_VALUE)
			if err == nil {
				key.Close()
				return errors.New("uninstall registration already exists")
			}
			if !errors.Is(err, registry.ErrNotExist) {
				return err
			}
			if err := safefs.RejectLinkedComponents(shortcut(record)); err != nil {
				return err
			}
			if _, err := os.Lstat(shortcut(record)); !errors.Is(err, os.ErrNotExist) {
				return errors.New("Start Menu shortcut already exists")
			}
			return nil
		},
		prepare: func(record Record) (registration, error) {
			return prepareWindowsRegistration(record, shortcuts, registryRoot)
		},
		validateRemoval: validate,
		unregister: func(record Record) error {
			if err := validate(record); err != nil {
				return err
			}
			if err := os.Remove(shortcut(record)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err := registry.DeleteKey(registry.CURRENT_USER, registryRoot+record.App.ID); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return err
			}
			_ = os.Remove(shortcuts)
			return nil
		},
		removable: func(path string) error {
			name, err := windows.UTF16PtrFromString(path)
			if err != nil {
				return err
			}
			handle, err := windows.CreateFile(name, windows.DELETE, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
			if err != nil {
				return err
			}
			return windows.CloseHandle(handle)
		},
	}
}

func prepareWindowsRegistration(record Record, shortcuts, registryRoot string) (registration, error) {
	if err := safefs.EnsureDirectory(shortcuts, 0o755); err != nil {
		return registration{}, err
	}
	temp, err := os.MkdirTemp(shortcuts, ".velox-link-")
	if err != nil {
		return registration{}, err
	}
	cleanup := func() { _ = os.RemoveAll(temp) }
	executable := filepath.Join(record.Directory, "app", record.App.ID+".exe")
	staged := filepath.Join(temp, "app.lnk")
	if err := createShortcut(staged, executable, record.App.Name); err != nil {
		cleanup()
		return registration{}, err
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		cleanup()
		return registration{}, err
	}
	digest := sha256.Sum256(data)
	return registration{
		shortcutHash: hex.EncodeToString(digest[:]),
		cleanup:      cleanup,
		publish: func() error {
			key, existing, err := registry.CreateKey(registry.CURRENT_USER, registryRoot+record.App.ID, registry.SET_VALUE)
			if err != nil {
				return err
			}
			if existing {
				key.Close()
				return errors.New("uninstall registration appeared during installation")
			}
			success := false
			defer func() {
				key.Close()
				if !success {
					_ = registry.DeleteKey(registry.CURRENT_USER, registryRoot+record.App.ID)
				}
			}()
			uninstaller := filepath.Join(record.Directory, "uninstall.exe")
			for name, value := range map[string]string{
				"DisplayName": record.App.Name, "DisplayVersion": record.App.Version,
				"InstallLocation": record.Directory, "DisplayIcon": executable,
				"UninstallString": fmt.Sprintf(`"%s" --uninstall %s`, uninstaller, record.App.ID),
				"VeloxInstaller":  stateSchema,
			} {
				if err := key.SetStringValue(name, value); err != nil {
					return err
				}
			}
			for _, name := range []string{"NoModify", "NoRepair"} {
				if err := key.SetDWordValue(name, 1); err != nil {
					return err
				}
			}
			if err := os.Link(staged, filepath.Join(shortcuts, record.App.ID+".lnk")); err != nil {
				return err
			}
			success = true
			return nil
		},
	}, nil
}
