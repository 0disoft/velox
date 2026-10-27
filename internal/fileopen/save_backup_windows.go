//go:build windows

package fileopen

import (
	"errors"

	"golang.org/x/sys/windows"
)

// Delete only the verified old file object. A concurrent replacement can make
// ReplaceFile back up a different file even after the final preflight check.
func discardSaveBackup(path string, original windows.Handle, expected windows.ByHandleFileInformation) (err error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES|windows.DELETE,
		windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, windows.CloseHandle(handle)) }()
	var actual windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &actual); err != nil {
		return err
	}
	var current windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(original, &current); err != nil {
		return err
	}
	if !matchesSaveBackup(actual, current, expected) {
		return ErrConflict
	}
	// Mark the same object just checked; a path-based Remove would reopen a race.
	deleteFile := byte(1)
	return windows.SetFileInformationByHandle(handle, windows.FileDispositionInfo, &deleteFile, 1)
}

func matchesSaveBackup(actual, current, expected windows.ByHandleFileInformation) bool {
	// Compare live handles after rename: FAT IDs can change when a name grows.
	// The backup handle denies other rename/delete opens during this check.
	return versionMetadata(actual) == versionMetadata(current) &&
		actual.FileSizeHigh == expected.FileSizeHigh && actual.FileSizeLow == expected.FileSizeLow &&
		actual.LastWriteTime == expected.LastWriteTime && actual.FileAttributes == expected.FileAttributes &&
		actual.NumberOfLinks == expected.NumberOfLinks
}
