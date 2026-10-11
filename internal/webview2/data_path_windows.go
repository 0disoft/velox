package webview2

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// Chromium creates storage files beneath this directory, which can exceed
// MAX_PATH even when the configured profile itself is shorter than that limit.
func browserDataPath(path string) string {
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\\.\`) || !filepath.IsAbs(path) {
		return path
	}
	path = filepath.Clean(path)
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + path[2:]
	}
	return `\\?\` + path
}

func resolveBrowserDataPath(path string) string {
	extended := browserDataPath(path)
	if !filepath.IsAbs(path) || strings.HasPrefix(extended, `\\?\UNC\`) || strings.HasPrefix(extended, `\\.\`) {
		return extended
	}
	pointer, err := windows.UTF16PtrFromString(extended)
	if err != nil {
		return extended
	}
	size, err := windows.GetShortPathName(pointer, nil, 0)
	if err != nil || size == 0 || size > 32768 {
		return extended
	}
	buffer := make([]uint16, size)
	length, err := windows.GetShortPathName(pointer, &buffer[0], size)
	if err != nil || length == 0 || length >= size {
		return extended
	}
	short := windows.UTF16ToString(buffer[:length])
	if len(short) >= len(extended) {
		return extended
	}
	// Use an existing alias only; never move or substitute the user's profile.
	original, err := os.Stat(extended)
	if err != nil || !original.IsDir() {
		return extended
	}
	alias, err := os.Stat(short)
	if err != nil || !os.SameFile(original, alias) {
		return extended
	}
	return browserDataPath(short)
}
