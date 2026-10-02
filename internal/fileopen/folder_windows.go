//go:build windows

package fileopen

import (
	"path/filepath"

	"github.com/0disoft/velox/internal/safefs"
	"golang.org/x/sys/windows"
)

func NewWindowsFolder(hwnd uintptr, dispatch func(func()), active func() bool, generation func() uint64) *Folder {
	valid := func() bool {
		window, _, _ := dialogUser32.NewProc("IsWindow").Call(hwnd)
		return window != 0 && active()
	}
	return &Folder{Dispatch: dispatch, Active: valid, Generation: generation,
		Choose: func() (string, error) { return selectDialog(hwnd, newFolderDialog) }, Inspect: inspectSelectedFolder}
}

func newFolderDialog() (*dialogObject, error) {
	// Pick one existing filesystem folder, without cwd/recent-item changes or shortcut resolution.
	return createTextDialog(&fileOpenClass, &fileOpenInterface, 0x02101868, "Select local folder (read-only)", "")
}

func openSelectedFolder(path string) (windows.Handle, FolderSnapshot, error) {
	if !localPath(path) || filepath.Clean(path) != path {
		return 0, FolderSnapshot{}, ErrFolderUnsupported
	}
	root, _ := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	switch windows.GetDriveType(root) {
	case windows.DRIVE_FIXED, windows.DRIVE_REMOVABLE, windows.DRIVE_RAMDISK:
	default:
		return 0, FolderSnapshot{}, ErrFolderUnsupported
	}
	if safefs.RejectLinkedComponents(path) != nil {
		return 0, FolderSnapshot{}, ErrFolderUnsupported
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, FolderSnapshot{}, ErrFolderUnsupported
	}
	handle, err := windows.CreateFile(name, windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return 0, FolderSnapshot{}, err
	}
	fail := func(err error) (windows.Handle, FolderSnapshot, error) {
		windows.CloseHandle(handle)
		return 0, FolderSnapshot{}, err
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return fail(err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 || info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE|windows.FILE_ATTRIBUTE_ENCRYPTED) != 0 {
		return fail(ErrFolderUnsupported)
	}
	final, err := selectedFinalPath(handle, path)
	if err != nil {
		return fail(ErrFolderUnsupported)
	}
	label := filepath.Base(final)
	if filepath.Dir(final) == final {
		label = "Volume root"
	}
	return handle, FolderSnapshot{Name: label, ID: DirectoryID{info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow}}, nil
}

func inspectSelectedFolder(path string) (FolderSnapshot, error) {
	handle, snapshot, err := openSelectedFolder(path)
	if err != nil {
		return FolderSnapshot{}, err
	}
	windows.CloseHandle(handle)
	return snapshot, nil
}
