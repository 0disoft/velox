package ipc

import (
	"encoding/json"
	"errors"

	"github.com/0disoft/velox/internal/fileopen"
)

const (
	PermissionFolderRead     = "folder.read"
	PermissionFolderReadText = "folder.readText"
)

type FolderAccess interface {
	Select(func(fileopen.FolderResult, error)) error
	List(uint32, func(fileopen.FolderListing, error)) error
	OpenText(uint32, string, func(fileopen.Result, error)) error
	ReleaseTarget(uint32) error
	ClearTarget()
}

func (d *Dispatcher) SetFolderAccess(access FolderAccess) {
	d.mu.Lock()
	d.folders = access
	d.mu.Unlock()
}

func (d *Dispatcher) DropFolderTarget() {
	d.mu.Lock()
	access := d.folders
	d.mu.Unlock()
	if access != nil {
		access.ClearTarget()
	}
}

func folderFailure(id uint32, err error) Response {
	code, message := "NATIVE_OPERATION_FAILED", "The folder operation failed."
	switch {
	case errors.Is(err, fileopen.ErrBusy):
		code, message = "TOO_MANY_REQUESTS", fileopen.ErrBusy.Error()
	case errors.Is(err, fileopen.ErrFolderUnsupported):
		code, message = "UNSUPPORTED_FOLDER", fileopen.ErrFolderUnsupported.Error()
	case errors.Is(err, fileopen.ErrFolderTarget):
		code, message = "FOLDER_TARGET_INVALID", fileopen.ErrFolderTarget.Error()
	case errors.Is(err, fileopen.ErrTooLarge):
		code, message = "PAYLOAD_TOO_LARGE", fileopen.ErrTooLarge.Error()
	case errors.Is(err, fileopen.ErrUnsupported):
		code, message = "UNSUPPORTED_FILE", fileopen.ErrUnsupported.Error()
	}
	return failure(id, code, message)
}

func folderTargetParam(raw json.RawMessage) (uint32, error) {
	var p struct {
		Target *uint32 `json:"target"`
	}
	if decodeSaveParams(raw, &p) != nil || p.Target == nil || *p.Target == 0 {
		return 0, errors.New("Folder parameters must contain only a positive target token.")
	}
	return *p.Target, nil
}

func (d *Dispatcher) releaseFolder(request Request) Response {
	target, err := folderTargetParam(request.Params)
	if err != nil {
		return failure(request.ID, "INVALID_PARAMS", err.Error())
	}
	d.mu.Lock()
	access := d.folders
	d.mu.Unlock()
	if access == nil {
		return folderFailure(request.ID, errors.New("unavailable"))
	}
	if err := access.ReleaseTarget(target); err != nil {
		return folderFailure(request.ID, err)
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
}

func (d *Dispatcher) selectFolder(request Request, finish func(Response)) {
	if _, granted := d.permissions[PermissionFolderRead]; !granted {
		finish(failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted."))
		return
	}
	if err := requireEmptyParams(request.Params); err != nil {
		finish(failure(request.ID, "INVALID_PARAMS", err.Error()))
		return
	}
	d.mu.Lock()
	access := d.folders
	d.mu.Unlock()
	if access == nil {
		finish(folderFailure(request.ID, errors.New("unavailable")))
		return
	}
	respond := func(result fileopen.FolderResult, err error) {
		if err != nil {
			finish(folderFailure(request.ID, err))
			return
		}
		finish(Response{Version: Version, ID: request.ID, OK: true, Result: result})
	}
	if err := access.Select(respond); err != nil {
		respond(fileopen.FolderResult{}, err)
	}
}

func (d *Dispatcher) listFolder(request Request, finish func(Response)) {
	if _, granted := d.permissions[PermissionFolderRead]; !granted {
		finish(failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted."))
		return
	}
	target, err := folderTargetParam(request.Params)
	if err != nil {
		finish(failure(request.ID, "INVALID_PARAMS", err.Error()))
		return
	}
	d.mu.Lock()
	access := d.folders
	d.mu.Unlock()
	if access == nil {
		finish(folderFailure(request.ID, errors.New("unavailable")))
		return
	}
	respond := func(result fileopen.FolderListing, err error) {
		if err != nil {
			finish(folderFailure(request.ID, err))
			return
		}
		finish(Response{Version: Version, ID: request.ID, OK: true, Result: result})
	}
	if err := access.List(target, respond); err != nil {
		respond(fileopen.FolderListing{}, err)
	}
}

func (d *Dispatcher) openFolderText(request Request, finish func(Response)) {
	for _, permission := range []string{PermissionFolderRead, PermissionFolderReadText} {
		if _, granted := d.permissions[permission]; !granted {
			finish(failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted."))
			return
		}
	}
	var p struct {
		Target *uint32 `json:"target"`
		Name   *string `json:"name"`
	}
	if decodeSaveParams(request.Params, &p) != nil || p.Target == nil || *p.Target == 0 || p.Name == nil || fileopen.ValidateSaveName(*p.Name) != nil {
		finish(failure(request.ID, "INVALID_PARAMS", "Folder text parameters must contain only a positive target token and a supported basename."))
		return
	}
	d.mu.Lock()
	access := d.folders
	d.mu.Unlock()
	if access == nil {
		finish(folderFailure(request.ID, errors.New("unavailable")))
		return
	}
	respond := func(result fileopen.Result, err error) {
		if err != nil {
			finish(folderFailure(request.ID, err))
			return
		}
		finish(Response{Version: Version, ID: request.ID, OK: true, Result: result})
	}
	if err := access.OpenText(*p.Target, *p.Name, respond); err != nil {
		respond(fileopen.Result{}, err)
	}
}
