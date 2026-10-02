//go:build windows

package fileopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/0disoft/velox/internal/safefs"
	"golang.org/x/sys/windows"
)

var replaceFile = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

func writeSelected(path, text string) (SaveResult, error) {
	return writeText(path, text, nil)
}

func writeConnected(path, text string, expected FileVersion) (SaveResult, error) {
	return writeText(path, text, &expected)
}

func validateSavePath(path string) error {
	if !localPath(path) || filepath.Clean(path) != path || ValidateSaveName(filepath.Base(path)) != nil {
		return ErrUnsupported
	}
	root, _ := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	switch windows.GetDriveType(root) {
	case windows.DRIVE_FIXED, windows.DRIVE_REMOVABLE, windows.DRIVE_RAMDISK:
	default:
		return ErrUnsupported
	}
	if err := safefs.RejectLinkedComponents(path); err != nil {
		return ErrUnsupported
	}
	return nil
}

func validateSaveHandle(handle windows.Handle, path string, info *windows.ByHandleFileInformation) error {
	if err := windows.GetFileInformationByHandle(handle, info); err != nil {
		return err
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE|windows.FILE_ATTRIBUTE_READONLY|windows.FILE_ATTRIBUTE_ENCRYPTED) != 0 || info.NumberOfLinks != 1 {
		return ErrUnsupported
	}
	var final [32768]uint16
	length, err := windows.GetFinalPathNameByHandle(handle, &final[0], uint32(len(final)), 0)
	if err != nil || length == 0 || length >= uint32(len(final)) {
		return ErrUnsupported
	}
	finalPath := windows.UTF16ToString(final[:length])
	if !strings.HasPrefix(finalPath, `\\?\`) {
		return ErrUnsupported
	}
	finalPath = strings.TrimPrefix(finalPath, `\\?\`)
	if strings.EqualFold(finalPath, path) {
		return nil
	}
	// DOS short names can differ from the handle's normalized long spelling.
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ErrUnsupported
	}
	var long [32768]uint16
	length, err = windows.GetLongPathName(name, &long[0], uint32(len(long)))
	if err != nil || length == 0 || length >= uint32(len(long)) || !strings.EqualFold(windows.UTF16ToString(long[:length]), finalPath) {
		return ErrUnsupported
	}
	return nil
}

func writeText(path, text string, expected *FileVersion) (SaveResult, error) {
	if err := ValidateSaveText(text); err != nil {
		return SaveResult{}, err
	}
	if err := validateSavePath(path); err != nil {
		return SaveResult{}, err
	}
	name, _ := windows.UTF16PtrFromString(path)
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	exists := err == nil
	if expected != nil && errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return SaveResult{}, ErrConflict
	}
	if !exists && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return SaveResult{}, err
	}
	var original windows.ByHandleFileInformation
	if exists {
		defer windows.CloseHandle(handle)
		if err := validateSaveHandle(handle, path, &original); err != nil {
			return SaveResult{}, err
		}
		if expected != nil {
			actual, err := versionOfHandle(handle, original)
			if err != nil {
				return SaveResult{}, err
			}
			if actual != *expected {
				return SaveResult{}, ErrConflict
			}
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".velox-save-*")
	if err != nil {
		return SaveResult{}, err
	}
	tempPath := temp.Name()
	retain := false
	defer func() {
		temp.Close()
		if !retain {
			os.Remove(tempPath)
		}
	}()
	if exists {
		// Apply the selected file's DACL before putting its text into a sibling file.
		sd, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			return SaveResult{}, err
		}
		dacl, _, err := sd.DACL()
		if err != nil {
			return SaveResult{}, err
		}
		if err := windows.SetNamedSecurityInfo(tempPath, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
			return SaveResult{}, err
		}
	}
	if _, err := temp.WriteString(text); err != nil {
		return SaveResult{}, err
	}
	if err := temp.Sync(); err != nil {
		return SaveResult{}, err
	}
	if err := temp.Close(); err != nil {
		return SaveResult{}, err
	}
	if err := safefs.RejectLinkedComponents(path); err != nil {
		return SaveResult{}, ErrUnsupported
	}
	from, _ := windows.UTF16PtrFromString(tempPath)
	if exists {
		// Recheck identity immediately before replacing; do not silently overwrite a changed entry.
		check, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return SaveResult{}, err
		}
		var current windows.ByHandleFileInformation
		err = windows.GetFileInformationByHandle(check, &current)
		windows.CloseHandle(check)
		if err != nil || versionMetadata(current) != versionMetadata(original) || current.FileAttributes != original.FileAttributes || current.NumberOfLinks != original.NumberOfLinks {
			return SaveResult{}, ErrUnsupported
		}
		backup, err := os.CreateTemp(filepath.Dir(path), ".velox-backup-*")
		if err != nil {
			return SaveResult{}, err
		}
		backupPath := backup.Name()
		if err := backup.Close(); err != nil {
			os.Remove(backupPath)
			return SaveResult{}, err
		}
		if err := os.Remove(backupPath); err != nil {
			return SaveResult{}, err
		}
		toBackup, _ := windows.UTF16PtrFromString(backupPath)
		ok, _, _ := replaceFile.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(toBackup)), 0, 0, 0)
		if ok == 0 {
			retain = true
			return SaveResult{}, ErrRecovery
		}
		if err := os.Remove(backupPath); err != nil {
			return SaveResult{}, ErrRecovery
		}
	} else {
		// No REPLACE_EXISTING: a file appearing after selection must not be overwritten.
		if err := windows.MoveFileEx(from, name, windows.MOVEFILE_WRITE_THROUGH); err != nil {
			return SaveResult{}, err
		}
	}
	return SaveResult{Name: filepath.Base(path), Bytes: len(text)}, nil
}
