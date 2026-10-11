package installer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

type removalFile struct {
	file    *os.File
	parents []windows.Handle
}

func openRemovalFile(path string, owned File) (*removalFile, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	locked := &removalFile{}
	fail := func(err error) (*removalFile, error) {
		return nil, errors.Join(err, locked.close())
	}
	// Pin ancestors from the volume down so an intervening rename or junction
	// cannot redirect the exclusive file open to a different directory.
	var parents []string
	for parent := filepath.Dir(absolute); ; parent = filepath.Dir(parent) {
		parents = append(parents, parent)
		if filepath.Dir(parent) == parent {
			break
		}
	}
	for i := len(parents) - 1; i >= 0; i-- {
		handle, err := openRemovalHandle(parents[i], windows.FILE_READ_ATTRIBUTES,
			windows.FILE_SHARE_READ, windows.FILE_FLAG_BACKUP_SEMANTICS)
		if err != nil {
			return fail(err)
		}
		locked.parents = append(locked.parents, handle)
		var info windows.ByHandleFileInformation
		if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
			return fail(err)
		}
		if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return fail(errors.New("preserving a linked or non-directory installation path"))
		}
	}
	handle, err := openRemovalHandle(absolute, windows.GENERIC_READ|windows.DELETE, 0, 0)
	if err != nil {
		return fail(err)
	}
	locked.file = os.NewFile(uintptr(handle), absolute)
	if locked.file == nil {
		windows.CloseHandle(handle)
		return fail(errors.New("could not own the removal file handle"))
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return fail(err)
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE) != 0 {
		return fail(errors.New("preserving a linked, offline or non-regular installed file"))
	}
	if err := verifyRemovalFile(locked.file, owned); err != nil {
		return fail(err)
	}
	return locked, nil
}

func openRemovalHandle(path string, access, share, flags uint32) (windows.Handle, error) {
	if !strings.HasPrefix(path, `\\?\`) {
		if strings.HasPrefix(path, `\\`) {
			path = `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
		} else {
			path = `\\?\` + path
		}
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	handle, err := windows.CreateFile(name, access, share, nil, windows.OPEN_EXISTING,
		flags|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return windows.InvalidHandle, &os.PathError{Op: "open for removal", Path: path, Err: err}
	}
	return handle, nil
}

func (file *removalFile) remove() error {
	// FILE_DISPOSITION_INFO.DeleteFile is a one-byte BOOLEAN. Mark the same
	// handle whose bytes were hashed, not the path that could be replaced.
	deleteFile := byte(1)
	return windows.SetFileInformationByHandle(windows.Handle(file.file.Fd()), windows.FileDispositionInfo, &deleteFile, 1)
}

func (file *removalFile) close() error {
	var err error
	if file.file != nil {
		err = file.file.Close()
		file.file = nil
	}
	for i := len(file.parents) - 1; i >= 0; i-- {
		err = errors.Join(err, windows.CloseHandle(file.parents[i]))
	}
	file.parents = nil
	return err
}
