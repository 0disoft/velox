//go:build windows

package installer

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

func acquireOperationKey(key string) (func(), error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("read installation user: %w", err)
	}
	digest := sha256.Sum256([]byte(user.User.Sid.String() + "\x00" + key))
	name, err := windows.UTF16PtrFromString(fmt.Sprintf(`Global\Velox.InstallOperation.%x`, digest))
	if err != nil {
		return nil, err
	}
	// Object lifetime is the lease, without thread-affine mutex ownership.
	// Global coordinates this user's installations across Windows sessions.
	handle, err := windows.CreateMutex(nil, false, name)
	if handle == 0 || err != nil {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			return nil, errOperationInProgress
		}
		return nil, fmt.Errorf("acquire installation operation: %w", err)
	}
	return func() { windows.CloseHandle(handle) }, nil
}
