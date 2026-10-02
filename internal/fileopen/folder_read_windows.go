//go:build windows

package fileopen

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func readFolderText(path string, identity DirectoryID, name string) (Result, error) {
	if err := ValidateSaveName(name); err != nil {
		return Result{}, err
	}
	directory, snapshot, err := openSelectedFolder(path)
	if err != nil {
		if errors.Is(err, ErrFolderUnsupported) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			return Result{}, ErrFolderTarget
		}
		return Result{}, err
	}
	defer windows.CloseHandle(directory)
	if snapshot.ID != identity {
		return Result{}, ErrFolderTarget
	}
	return readFolderTextAt(directory, name)
}

func readFolderTextAt(directory windows.Handle, name string) (Result, error) {
	if err := ValidateSaveName(name); err != nil {
		return Result{}, err
	}
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return Result{}, ErrUnsupported
	}
	attributes := windows.OBJECT_ATTRIBUTES{
		RootDirectory: directory, ObjectName: objectName,
		Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
	}
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	var handle windows.Handle
	// Resolve only this basename against the checked directory handle, never its path.
	err = windows.NtCreateFile(&handle, windows.FILE_GENERIC_READ, &attributes,
		&windows.IO_STATUS_BLOCK{}, nil, 0, windows.FILE_SHARE_READ, windows.FILE_OPEN,
		windows.FILE_SYNCHRONOUS_IO_NONALERT|windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_OPEN_NO_RECALL, 0, 0)
	if err != nil {
		switch err {
		case windows.STATUS_FILE_IS_A_DIRECTORY, windows.STATUS_REPARSE_POINT_ENCOUNTERED,
			windows.STATUS_STOPPED_ON_SYMLINK, windows.STATUS_IO_REPARSE_TAG_NOT_HANDLED, windows.STATUS_REPARSE_POINT_NOT_RESOLVED:
			return Result{}, ErrUnsupported
		}
		return Result{}, err
	}
	file := os.NewFile(uintptr(handle), "selected-folder-file")
	if file == nil {
		windows.CloseHandle(handle)
		return Result{}, ErrUnsupported
	}
	defer file.Close()
	kind, err := windows.GetFileType(handle)
	if err != nil || kind != windows.FILE_TYPE_DISK {
		return Result{}, ErrUnsupported
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return Result{}, err
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE|windows.FILE_ATTRIBUTE_ENCRYPTED) != 0 || info.NumberOfLinks != 1 {
		return Result{}, ErrUnsupported
	}
	return readTextFile(file, name, info)
}
