package ipc

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/0disoft/velox/internal/fileopen"
)

const PermissionFileOpen = "file.open"

type FileOpener interface {
	Open(func(fileopen.Result, error)) error
}

func RequiresDeferred(raw json.RawMessage) bool {
	request, err := decodeRequest(raw)
	return err == nil && (request.Method == "file.openText" || request.Method == "file.commitSave")
}

func (d *Dispatcher) SetFileOpener(opener FileOpener) { d.mu.Lock(); d.files = opener; d.mu.Unlock() }

// DispatchAsync keeps the ID reserved until native selection completes.
func (d *Dispatcher) DispatchAsync(raw json.RawMessage, reply func(Response)) {
	request, err := decodeRequest(raw)
	if err != nil {
		reply(failure(request.ID, err.Code, err.Message))
		return
	}
	if err := d.begin(request.ID); err != nil {
		reply(failure(request.ID, err.Code, err.Message))
		return
	}
	var once sync.Once
	finish := func(response Response) {
		once.Do(func() {
			if d.IsClosing() {
				response = failure(request.ID, "SHUTTING_DOWN", "The native host is shutting down.")
			}
			d.finish(request.ID)
			reply(response)
		})
	}
	if request.Method == "file.commitSave" {
		d.commitSave(request, finish)
		return
	}
	if request.Method != "file.openText" {
		finish(d.dispatch(request))
		return
	}
	if _, granted := d.permissions[PermissionFileOpen]; !granted {
		finish(failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted."))
		return
	}
	if err := requireEmptyParams(request.Params); err != nil {
		finish(failure(request.ID, "INVALID_PARAMS", err.Error()))
		return
	}
	d.mu.Lock()
	opener := d.files
	d.mu.Unlock()
	if opener == nil {
		finish(failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed."))
		return
	}
	respond := func(result fileopen.Result, err error) {
		if err != nil {
			code, message := "NATIVE_OPERATION_FAILED", "The selected file could not be opened."
			switch {
			case errors.Is(err, fileopen.ErrBusy):
				code, message = "TOO_MANY_REQUESTS", fileopen.ErrBusy.Error()
			case errors.Is(err, fileopen.ErrTooLarge):
				code, message = "PAYLOAD_TOO_LARGE", fileopen.ErrTooLarge.Error()
			case errors.Is(err, fileopen.ErrUnsupported):
				code, message = "UNSUPPORTED_FILE", fileopen.ErrUnsupported.Error()
			}
			finish(failure(request.ID, code, message))
			return
		}
		finish(Response{Version: Version, ID: request.ID, OK: true, Result: result})
	}
	if err := opener.Open(respond); err != nil {
		respond(fileopen.Result{}, err)
	}
}
