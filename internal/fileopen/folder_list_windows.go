//go:build windows

package fileopen

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func listSelectedFolder(path string, identity DirectoryID) (FolderListing, error) {
	handle, snapshot, err := openSelectedFolder(path)
	if err != nil {
		if errors.Is(err, ErrFolderUnsupported) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			return FolderListing{}, ErrFolderTarget
		}
		return FolderListing{}, err
	}
	if snapshot.ID != identity {
		windows.CloseHandle(handle)
		return FolderListing{}, ErrFolderTarget
	}
	file := os.NewFile(uintptr(handle), "selected-folder")
	if file == nil {
		windows.CloseHandle(handle)
		return FolderListing{}, ErrFolderUnsupported
	}
	defer file.Close()
	// Go's Windows ReadDir enumerates this verified handle, not child paths.
	entries, err := file.ReadDir(MaxFolderEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return FolderListing{}, err
	}
	result := FolderListing{Entries: []FolderEntry{}, Truncated: len(entries) > MaxFolderEntries}
	if len(entries) > MaxFolderEntries {
		entries = entries[:MaxFolderEntries]
	}
	// Reserve more than the fixed JSON envelope plus bounded skipped count.
	encodedBytes := 128
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return FolderListing{}, err
		}
		attributes, ok := info.Sys().(*syscall.Win32FileAttributeData)
		if !ok || attributes.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE|windows.FILE_ATTRIBUTE_ENCRYPTED) != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			result.Skipped++
			continue
		}
		kind := "file"
		if info.IsDir() {
			kind = "directory"
		}
		item := FolderEntry{Name: entry.Name(), Kind: kind}
		encoded, err := json.Marshal(item)
		if err != nil {
			return FolderListing{}, err
		}
		if encodedBytes+len(encoded)+1 > MaxFolderResultBytes {
			result.Truncated = true
			break
		}
		encodedBytes += len(encoded) + 1
		result.Entries = append(result.Entries, item)
	}
	return result, nil
}
