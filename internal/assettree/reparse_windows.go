//go:build windows

package assettree

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func isLinkOrReparse(path string, info os.FileInfo) (bool, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, fmt.Errorf("encode asset path: %w", err)
	}
	attributes, err := windows.GetFileAttributes(pointer)
	if err != nil {
		return false, fmt.Errorf("read asset attributes: %w", err)
	}
	return attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}
