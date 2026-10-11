package webview2

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const maxProfileLinkPath = 120

type profileLink struct {
	path     string
	handles  []windows.Handle
	once     sync.Once
	closeErr error
}

func (link *profileLink) Close() error {
	link.once.Do(func() {
		for i := len(link.handles) - 1; i >= 0; i-- {
			link.closeErr = errors.Join(link.closeErr, windows.CloseHandle(link.handles[i]))
		}
		link.handles = nil
	})
	return link.closeErr
}

// Links persist after handle release: the browser can outlive its GUI host.
// This function never moves a profile or recursively deletes a directory.
func acquireProfileLink(profile, root string) (*profileLink, error) {
	profile, err := localProfilePath(profile)
	if err != nil {
		return nil, err
	}
	root, err = localProfilePath(root)
	if err != nil {
		return nil, err
	}
	if relative, err := filepath.Rel(profile, root); err != nil || relative == "." || (relative != ".." && !strings.HasPrefix(relative, `..\`)) {
		return nil, errors.New("profile link root must stay outside the profile")
	}
	link := &profileLink{}
	fail := func(err error) (*profileLink, error) { return nil, errors.Join(err, link.Close()) }
	profileHandle, err := link.pinPlainDirectories(profile, false)
	if err != nil {
		return fail(fmt.Errorf("pin profile ancestors: %w", err))
	}
	var identity windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(profileHandle, &identity); err != nil {
		return fail(err)
	}
	target, err := finalProfilePath(profileHandle)
	if err != nil {
		return fail(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fail(err)
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%08x|%08x%08x|%s", user.User.Sid.String(),
		identity.VolumeSerialNumber, identity.FileIndexHigh, identity.FileIndexLow, strings.ToLower(target))))
	name := fmt.Sprintf("%x", digest[:16])
	link.path = browserDataPath(filepath.Join(root, name))
	spelling, err := windows.UTF16FromString(link.path)
	if err != nil || len(spelling)-1 > maxProfileLinkPath {
		return fail(errors.New("profile link path is too long"))
	}
	if _, err := link.pinPlainDirectories(filepath.Dir(root), false); err != nil {
		return fail(fmt.Errorf("pin link root ancestors: %w", err))
	}
	descriptor, err := windows.SecurityDescriptorFromString("O:" + user.User.Sid.String() + "D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")(A;OICI;FA;;;SY)")
	if err != nil {
		return fail(err)
	}
	attributes := windows.SecurityAttributes{SecurityDescriptor: descriptor}
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	rootName, err := windows.UTF16PtrFromString(browserDataPath(root))
	if err != nil {
		return fail(err)
	}
	if err := windows.CreateDirectory(rootName, &attributes); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return fail(fmt.Errorf("create private link root: %w", err))
	}
	rootHandle, err := link.pinPlainDirectories(root, true)
	if err != nil {
		return fail(fmt.Errorf("pin private link root: %w", err))
	}
	if err := verifyProfileLinkRoot(rootHandle, descriptor); err != nil {
		return fail(err)
	}
	alias, err := openProfileLinkChild(rootHandle, name, windows.FILE_OPEN, false)
	if errors.Is(err, windows.STATUS_OBJECT_NAME_NOT_FOUND) {
		if err := publishProfileLink(rootHandle, name, target); err != nil {
			return fail(fmt.Errorf("publish profile link: %w", err))
		}
		alias, err = openProfileLinkChild(rootHandle, name, windows.FILE_OPEN, false)
	}
	if err != nil {
		return fail(fmt.Errorf("open published profile link: %w", err))
	}
	link.handles = append(link.handles, alias)
	var rootInfo windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(rootHandle, &rootInfo); err != nil {
		return fail(err)
	}
	if rootInfo.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fail(errors.New("profile link root became a linked directory"))
	}
	actual, err := profileJunctionTarget(alias)
	if err != nil || !strings.EqualFold(actual, `\??\`+strings.TrimPrefix(target, `\\?\`)) {
		return fail(errors.New("profile link is not the expected directory junction"))
	}
	// Verify the followed object too, while the junction and all ancestors are pinned.
	aliasName, _ := windows.UTF16PtrFromString(link.path)
	followed, err := windows.CreateFile(aliasName, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return fail(err)
	}
	link.handles = append(link.handles, followed)
	var observed windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(followed, &observed); err != nil {
		return fail(err)
	}
	if identity.VolumeSerialNumber != observed.VolumeSerialNumber || identity.FileIndexHigh != observed.FileIndexHigh || identity.FileIndexLow != observed.FileIndexLow {
		return fail(errors.New("profile link changed directory identity"))
	}
	return link, nil
}

func localProfilePath(path string) (string, error) {
	path = strings.TrimPrefix(path, `\\?\`)
	if !filepath.IsAbs(path) || len(filepath.VolumeName(path)) != 2 {
		return "", errors.New("profile links require absolute local drive paths")
	}
	path = filepath.Clean(path)
	if _, err := windows.UTF16FromString(path); err != nil {
		return "", errors.New("invalid profile link path")
	}
	drive, _ := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if windows.GetDriveType(drive) != windows.DRIVE_FIXED {
		return "", errors.New("profile links require a fixed local drive")
	}
	return path, nil
}

func (link *profileLink) pinPlainDirectories(path string, writableLeaf bool) (windows.Handle, error) {
	var parents []string
	for current := path; ; current = filepath.Dir(current) {
		parents = append(parents, current)
		if filepath.Dir(current) == current {
			break
		}
	}
	var last windows.Handle
	for i := len(parents) - 1; i >= 0; i-- {
		name, err := windows.UTF16PtrFromString(browserDataPath(parents[i]))
		if err != nil {
			return 0, err
		}
		share := uint32(windows.FILE_SHARE_READ)
		if i > 0 || writableLeaf {
			share |= windows.FILE_SHARE_WRITE
		}
		handle, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES|windows.FILE_LIST_DIRECTORY|windows.READ_CONTROL,
			share, nil, windows.OPEN_EXISTING,
			windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return 0, err
		}
		link.handles = append(link.handles, handle)
		var info windows.ByHandleFileInformation
		if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
			return 0, err
		}
		if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 || info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_OFFLINE) != 0 {
			return 0, errors.New("linked or offline profile link ancestor is not supported")
		}
		last = handle
	}
	return last, nil
}

func finalProfilePath(handle windows.Handle) (string, error) {
	buffer := make([]uint16, 32768)
	length, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
	if err != nil {
		return "", err
	}
	if length == 0 || length >= uint32(len(buffer)) {
		return "", errors.New("profile target path exceeds the supported length")
	}
	path := windows.UTF16ToString(buffer[:length])
	if _, err := localProfilePath(path); err != nil {
		return "", err
	}
	return path, nil
}

func verifyProfileLinkRoot(handle windows.Handle, expected *windows.SECURITY_DESCRIPTOR) error {
	actual, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := actual.Owner()
	if err != nil {
		return err
	}
	wantOwner, _, err := expected.Owner()
	if err != nil || owner == nil || wantOwner == nil || !windows.EqualSid(owner, wantOwner) {
		return errors.New("profile link root has an unexpected owner")
	}
	control, _, err := actual.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return errors.New("profile link root permissions are not protected")
	}
	got, _, err := actual.DACL()
	if err != nil {
		return err
	}
	want, _, err := expected.DACL()
	if err != nil || got == nil || want == nil {
		return errors.New("profile link root permissions are missing")
	}
	// Compare canonical SDDL DACLs, ignoring unrelated security-descriptor fields.
	gotSD, _ := windows.NewSecurityDescriptor()
	wantSD, _ := windows.NewSecurityDescriptor()
	if err := gotSD.SetDACL(got, true, false); err != nil {
		return err
	}
	if err := wantSD.SetDACL(want, true, false); err != nil {
		return err
	}
	if gotSD.String() != wantSD.String() {
		return errors.New("profile link root grants unexpected access")
	}
	return nil
}

func openProfileLinkChild(root windows.Handle, name string, disposition uint32, writable bool) (windows.Handle, error) {
	if name == "" || strings.ContainsAny(name, `\/:`) {
		return 0, errors.New("invalid profile link child name")
	}
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return 0, err
	}
	attributes := windows.OBJECT_ATTRIBUTES{RootDirectory: root, ObjectName: objectName, Attributes: windows.OBJ_CASE_INSENSITIVE}
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	access := uint32(windows.FILE_READ_ATTRIBUTES | windows.FILE_LIST_DIRECTORY | windows.SYNCHRONIZE)
	if writable {
		access |= windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.DELETE
	}
	var status windows.IO_STATUS_BLOCK
	var handle windows.Handle
	deadline := time.Now().Add(2 * time.Second)
	for {
		err = windows.NtCreateFile(&handle, access, &attributes, &status, nil, windows.FILE_ATTRIBUTE_DIRECTORY,
			windows.FILE_SHARE_READ, disposition, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
		if !errors.Is(err, windows.STATUS_SHARING_VIOLATION) || time.Now().After(deadline) {
			return handle, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func publishProfileLink(root windows.Handle, name, target string) (err error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	stage, err := openProfileLinkChild(root, fmt.Sprintf("tmp-%x", random), windows.FILE_CREATE, true)
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	published := false
	defer func() {
		if !published {
			remove := byte(1)
			err = errors.Join(err, windows.SetFileInformationByHandle(stage, windows.FileDispositionInfo, &remove, 1))
		}
		err = errors.Join(err, windows.CloseHandle(stage))
	}()
	if err = setProfileJunctionTarget(stage, target); err != nil {
		return fmt.Errorf("set staging mount point: %w", err)
	}
	if err = renameProfileLink(stage, root, name); errors.Is(err, windows.STATUS_OBJECT_NAME_COLLISION) {
		return nil // Another publisher won. Remove only this unopened staging object.
	}
	if err == nil {
		published = true
	}
	if err != nil {
		return fmt.Errorf("publish staging name: %w", err)
	}
	return err
}
