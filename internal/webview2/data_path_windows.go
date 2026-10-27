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

func prepareBrowserProfilePath(profile string) (string, *profileLink, error) {
	return prepareProfilePath(profile, resolveBrowserDataPath, os.UserCacheDir)
}

func prepareProfilePath(profile string, resolve func(string) string, cacheDir func() (string, error)) (string, *profileLink, error) {
	path := resolve(profile)
	spelling, err := windows.UTF16FromString(path)
	if err != nil {
		return "", nil, ErrProfileUnavailable
	}
	// Keep the existing path for short profiles and unsupported drive types.
	// The budget reserves room for Chromium's nested storage filenames.
	if len(spelling)-1 <= maxProfileLinkPath {
		return shortBrowserProfilePath(path), nil, nil
	}
	if _, err := localProfilePath(profile); err != nil {
		return path, nil, nil
	}
	base, err := cacheDir()
	if err != nil {
		return "", nil, ErrProfileUnavailable
	}
	link, err := acquireProfileLink(profile, filepath.Join(base, "VeloxProfileLinks"))
	if err != nil {
		// Neither a failed link nor a known-long fallback should start a new
		// browser profile. Do not expose paths or native errors in startup output.
		return "", nil, ErrProfileUnavailable
	}
	return shortBrowserProfilePath(link.path), link, nil
}

// WebView2 92 cannot open IndexedDB with an extended-length profile spelling.
// The caller has already bounded this path; strip the prefix only after proving
// the DOS spelling names the same existing directory. Native file operations
// and the pinned junction lease retain their extended-length paths.
func shortBrowserProfilePath(path string) string {
	plain, extended := strings.CutPrefix(path, `\\?\`)
	if !extended || !filepath.IsAbs(plain) || len(filepath.VolumeName(plain)) != 2 {
		return path
	}
	original, err := os.Stat(path)
	if err != nil || !original.IsDir() {
		return path
	}
	alias, err := os.Stat(plain)
	if err != nil || !os.SameFile(original, alias) {
		return path
	}
	return plain
}
