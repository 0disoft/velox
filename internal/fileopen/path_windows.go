//go:build windows

package fileopen

import (
	"strings"

	"golang.org/x/sys/windows"
)

func selectedFinalPath(handle windows.Handle, path string) (string, error) {
	var final [32768]uint16
	length, err := windows.GetFinalPathNameByHandle(handle, &final[0], uint32(len(final)), 0)
	if err != nil || length == 0 || length >= uint32(len(final)) {
		return "", ErrUnsupported
	}
	finalPath := windows.UTF16ToString(final[:length])
	if !strings.HasPrefix(finalPath, `\\?\`) {
		return "", ErrUnsupported
	}
	finalPath = strings.TrimPrefix(finalPath, `\\?\`)
	if strings.EqualFold(finalPath, path) {
		return finalPath, nil
	}
	// DOS short names can differ from the handle's normalized long spelling.
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", ErrUnsupported
	}
	var long [32768]uint16
	length, err = windows.GetLongPathName(name, &long[0], uint32(len(long)))
	if err != nil || length == 0 || length >= uint32(len(long)) || !strings.EqualFold(windows.UTF16ToString(long[:length]), finalPath) {
		return "", ErrUnsupported
	}
	return finalPath, nil
}
