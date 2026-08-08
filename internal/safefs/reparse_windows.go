//go:build windows

package safefs

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func IsLinkOrReparse(value string, info os.FileInfo) (bool, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	pointer, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return false, fmt.Errorf("encode path: %w", err)
	}
	attributes, err := windows.GetFileAttributes(pointer)
	if err != nil {
		return false, fmt.Errorf("read path attributes: %w", err)
	}
	return attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}
