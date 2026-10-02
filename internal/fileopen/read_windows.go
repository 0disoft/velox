//go:build windows

package fileopen

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/velox/internal/safefs"
	"golang.org/x/sys/windows"
)

func localPath(path string) bool {
	if len(path) < 3 || path[1] != ':' || (path[2] != '\\' && path[2] != '/') || !((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) || strings.Contains(path[2:], ":") {
		return false
	}
	for _, char := range path {
		if char < 32 {
			return false
		}
	}
	return true
}

func readSelected(path string) (Result, error) {
	if !localPath(path) {
		return Result{}, ErrUnsupported
	}
	root, _ := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	switch windows.GetDriveType(root) {
	case windows.DRIVE_FIXED, windows.DRIVE_REMOVABLE, windows.DRIVE_CDROM, windows.DRIVE_RAMDISK:
	default:
		return Result{}, ErrUnsupported
	}
	if err := safefs.RejectLinkedComponents(path); err != nil {
		return Result{}, ErrUnsupported
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return Result{}, ErrUnsupported
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return Result{}, err
	}
	file := os.NewFile(uintptr(handle), "selected-file")
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
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE) != 0 {
		return Result{}, ErrUnsupported
	}
	var final [32768]uint16
	length, err := windows.GetFinalPathNameByHandle(handle, &final[0], uint32(len(final)), 0)
	if err != nil || length == 0 || length >= uint32(len(final)) {
		return Result{}, ErrUnsupported
	}
	finalPath := windows.UTF16ToString(final[:length])
	if !strings.HasPrefix(finalPath, `\\?\`) || !localPath(strings.TrimPrefix(finalPath, `\\?\`)) {
		return Result{}, ErrUnsupported
	}
	return readTextFile(file, filepath.Base(finalPath), info)
}

func readTextFile(file *os.File, name string, info windows.ByHandleFileInformation) (Result, error) {
	if info.FileSizeHigh != 0 || info.FileSizeLow > MaxTextBytes {
		return Result{}, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxTextBytes+1))
	if err != nil {
		return Result{}, err
	}
	if len(data) > MaxTextBytes {
		return Result{}, ErrTooLarge
	}
	text := bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(text) || bytes.IndexByte(text, 0) >= 0 {
		return Result{}, ErrUnsupported
	}
	return Result{Name: name, Text: string(text), Bytes: len(data)}, nil
}
