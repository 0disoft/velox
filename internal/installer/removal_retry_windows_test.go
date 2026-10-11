package installer

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsDirectoryLockKeepsRemovalRetryRecord(t *testing.T) {
	for _, relative := range []string{"app/web", ""} {
		t.Run(relative, func(t *testing.T) {
			source, setup, env := fixture(t)
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(installed.Directory, filepath.FromSlash(relative))
			handle := windows.InvalidHandle
			t.Cleanup(func() {
				if handle != windows.InvalidHandle {
					_ = windows.CloseHandle(handle)
				}
			})
			env.unregister = func(Record) error {
				name, err := windows.UTF16PtrFromString(path)
				if err != nil {
					return err
				}
				handle, err = windows.CreateFile(name, windows.GENERIC_READ,
					windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING,
					windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
				return err
			}
			if _, err := uninstall(installed.AppID, env); err == nil {
				t.Fatal("removal ignored a real directory deletion lock")
			}
			if _, err := readState(installed.Directory, installed.AppID); err != nil {
				t.Fatalf("late directory lock lost its retry record: %v", err)
			}
			if err := windows.CloseHandle(handle); err != nil {
				t.Fatal(err)
			}
			handle = windows.InvalidHandle
			env.unregister = func(Record) error { return nil }
			if _, err := uninstall(installed.AppID, env); err != nil {
				t.Fatalf("cannot retry after directory unlock: %v", err)
			}
			if _, err := os.Stat(installed.Directory); !os.IsNotExist(err) {
				t.Fatalf("installation remains: %v", err)
			}
		})
	}
}
