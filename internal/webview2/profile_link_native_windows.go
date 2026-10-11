package webview2

import (
	"encoding/binary"
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func setProfileJunctionTarget(handle windows.Handle, target string) error {
	printName, err := windows.UTF16FromString(strings.TrimPrefix(target, `\\?\`))
	if err != nil {
		return err
	}
	substitute, err := windows.UTF16FromString(`\??\` + strings.TrimPrefix(target, `\\?\`))
	if err != nil {
		return err
	}
	size := 16 + 2*(len(substitute)+len(printName))
	if size > 16384 {
		return errors.New("profile junction target is too long")
	}
	data := make([]byte, size)
	binary.LittleEndian.PutUint32(data, windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(data[4:], uint16(size-8))
	binary.LittleEndian.PutUint16(data[10:], uint16(2*(len(substitute)-1)))
	binary.LittleEndian.PutUint16(data[12:], uint16(2*len(substitute)))
	binary.LittleEndian.PutUint16(data[14:], uint16(2*(len(printName)-1)))
	for i, character := range append(substitute, printName...) {
		binary.LittleEndian.PutUint16(data[16+i*2:], character)
	}
	var returned uint32
	return windows.DeviceIoControl(handle, windows.FSCTL_SET_REPARSE_POINT, &data[0], uint32(len(data)), nil, 0, &returned, nil)
}

func profileJunctionTarget(handle windows.Handle) (string, error) {
	data := make([]byte, 16384)
	var returned uint32
	if err := windows.DeviceIoControl(handle, windows.FSCTL_GET_REPARSE_POINT, nil, 0, &data[0], uint32(len(data)), &returned, nil); err != nil {
		return "", err
	}
	if returned < 16 || returned > uint32(len(data)) || binary.LittleEndian.Uint32(data) != windows.IO_REPARSE_TAG_MOUNT_POINT {
		return "", errors.New("profile link is not a mount point")
	}
	data = data[:returned]
	if int(binary.LittleEndian.Uint16(data[4:]))+8 != len(data) {
		return "", errors.New("invalid mount-point data length")
	}
	offset, length := int(binary.LittleEndian.Uint16(data[8:])), int(binary.LittleEndian.Uint16(data[10:]))
	if offset%2 != 0 || length%2 != 0 || length == 0 || offset+length > len(data)-16 {
		return "", errors.New("invalid mount-point target bounds")
	}
	characters := make([]uint16, length/2)
	for i := range characters {
		characters[i] = binary.LittleEndian.Uint16(data[16+offset+i*2:])
		if characters[i] == 0 {
			return "", errors.New("invalid mount-point target terminator")
		}
	}
	return windows.UTF16ToString(characters), nil
}

func renameProfileLink(handle, root windows.Handle, name string) error {
	characters, err := windows.UTF16FromString(name)
	if err != nil {
		return err
	}
	characters = characters[:len(characters)-1]
	type renameInfo struct {
		Replace uint32
		Root    windows.Handle
		Length  uint32
		Name    [1]uint16
	}
	var header renameInfo
	offset := int(unsafe.Offsetof(header.Name))
	data := make([]byte, offset+len(characters)*2)
	info := (*renameInfo)(unsafe.Pointer(&data[0]))
	info.Root = root
	info.Length = uint32(len(characters) * 2)
	for i, character := range characters {
		binary.LittleEndian.PutUint16(data[offset+i*2:], character)
	}
	var status windows.IO_STATUS_BLOCK
	// Replace remains false; never overwrite an occupied final link name.
	return windows.NtSetInformationFile(handle, &status, &data[0], uint32(len(data)), windows.FileRenameInformation)
}
