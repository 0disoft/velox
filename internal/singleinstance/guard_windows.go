//go:build windows

package singleinstance

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

type Guard struct {
	handle    windows.Handle
	key       string
	primary   bool
	closeOnce sync.Once
}

func Acquire(appID, profile string) (*Guard, bool, error) {
	canonical, err := canonicalProfile(profile)
	if err != nil {
		return nil, false, err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, false, fmt.Errorf("read single-instance user: %w", err)
	}
	var session uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &session); err != nil {
		return nil, false, fmt.Errorf("read single-instance session: %w", err)
	}
	key := identityKey(user.User.Sid.String(), session, appID, canonical)
	name, _ := windows.UTF16PtrFromString(`Local\` + key)
	// The object's lifetime is the lease; do not take thread-affine mutex ownership.
	handle, err := windows.CreateMutex(nil, false, name)
	if handle == 0 || (err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS)) {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		return nil, false, fmt.Errorf("acquire single-instance mutex: %w", err)
	}
	primary := !errors.Is(err, windows.ERROR_ALREADY_EXISTS)
	return &Guard{handle: handle, key: key, primary: primary}, primary, nil
}

func (g *Guard) Close() {
	g.closeOnce.Do(func() { windows.CloseHandle(g.handle) })
}

func canonicalProfile(profile string) (string, error) {
	if !filepath.IsAbs(profile) {
		return "", errors.New("single-instance profile must be absolute")
	}
	path := filepath.Clean(profile)
	var tail []string
	for {
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return "", fmt.Errorf("resolve single-instance profile: %w", err)
		}
		handle, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
		if err == nil {
			defer windows.CloseHandle(handle)
			var info windows.ByHandleFileInformation
			if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
				return "", fmt.Errorf("read single-instance profile attributes: %w", err)
			}
			if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
				return "", errors.New("single-instance profile ancestor is not a directory")
			}
			buffer := make([]uint16, 32768)
			n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
			if err != nil || n >= uint32(len(buffer)) {
				return "", fmt.Errorf("resolve final single-instance profile path: length %d, %v", n, err)
			}
			path = windows.UTF16ToString(buffer[:n])
			if strings.HasPrefix(path, `\\?\UNC\`) {
				path = `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
			} else {
				path = strings.TrimPrefix(path, `\\?\`)
			}
			for i := len(tail) - 1; i >= 0; i-- {
				path = filepath.Join(path, tail[i])
			}
			return strings.ToLower(filepath.Clean(path)), nil
		}
		if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) && !errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			return "", fmt.Errorf("open single-instance profile ancestor: %w", err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", errors.New("single-instance profile has no existing ancestor")
		}
		tail = append(tail, filepath.Base(path))
		path = parent
	}
}
