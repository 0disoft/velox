//go:build windows

package fileopen

import (
	"crypto/sha256"
	"errors"

	"golang.org/x/sys/windows"
)

func versionMetadata(info windows.ByHandleFileInformation) FileVersion {
	return FileVersion{Volume: info.VolumeSerialNumber, IndexHigh: info.FileIndexHigh, IndexLow: info.FileIndexLow,
		Size: uint64(info.FileSizeHigh)<<32 | uint64(info.FileSizeLow), Modified: uint64(info.LastWriteTime.HighDateTime)<<32 | uint64(info.LastWriteTime.LowDateTime)}
}

func versionOfHandle(handle windows.Handle, info windows.ByHandleFileInformation) (FileVersion, error) {
	version := versionMetadata(info)
	if version.Size > MaxTextBytes {
		return FileVersion{}, ErrConflict
	}
	hash := sha256.New()
	var buffer [32768]byte
	total := uint64(0)
	for {
		var read uint32
		err := windows.ReadFile(handle, buffer[:], &read, nil)
		if err != nil && !errors.Is(err, windows.ERROR_HANDLE_EOF) {
			return FileVersion{}, err
		}
		if read == 0 {
			break
		}
		total += uint64(read)
		if total > MaxTextBytes {
			return FileVersion{}, ErrConflict
		}
		hash.Write(buffer[:read])
	}
	var after windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &after); err != nil {
		return FileVersion{}, err
	}
	if total != version.Size || versionMetadata(after) != version {
		return FileVersion{}, ErrConflict
	}
	copy(version.Digest[:], hash.Sum(nil))
	return version, nil
}

func snapshotSelected(path string) (FileVersion, error) {
	if err := validateSavePath(path); err != nil {
		return FileVersion{}, err
	}
	name, _ := windows.UTF16PtrFromString(path)
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return FileVersion{}, err
	}
	defer windows.CloseHandle(handle)
	var info windows.ByHandleFileInformation
	if err := validateSaveHandle(handle, path, &info); err != nil {
		return FileVersion{}, err
	}
	return versionOfHandle(handle, info)
}
