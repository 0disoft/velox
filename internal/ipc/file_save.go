package ipc

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"github.com/0disoft/velox/internal/fileopen"
)

const PermissionFileSave = "file.save"

type FileSaver interface {
	Save(string, string, func(fileopen.SaveResult, error)) error
}

type saveUpload struct {
	token    uint32
	name     string
	expected int
	text     []byte
}

func (d *Dispatcher) SetFileSaver(saver FileSaver) {
	d.mu.Lock()
	d.saver = saver
	d.mu.Unlock()
}

// Discard staged text at navigation or shutdown, including abandoned uploads.
func (d *Dispatcher) DropPreparedText() {
	d.mu.Lock()
	d.upload = nil
	d.mu.Unlock()
}

func decodeSaveParams(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (d *Dispatcher) prepareSave(request Request) Response {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closing {
		return failure(request.ID, "SHUTTING_DOWN", "The native host is shutting down.")
	}
	invalid := func() Response { return failure(request.ID, "INVALID_PARAMS", "The save parameters are invalid.") }
	switch request.Method {
	case "file.beginSave":
		var p struct {
			Name  *string `json:"name"`
			Bytes *int    `json:"bytes"`
		}
		if decodeSaveParams(request.Params, &p) != nil || p.Name == nil || p.Bytes == nil || *p.Bytes < 0 || fileopen.ValidateSaveName(*p.Name) != nil {
			return invalid()
		}
		if *p.Bytes > fileopen.MaxTextBytes {
			return failure(request.ID, "PAYLOAD_TOO_LARGE", fileopen.ErrTooLarge.Error())
		}
		if d.saver == nil {
			return failure(request.ID, "NATIVE_OPERATION_FAILED", "Native saving is unavailable.")
		}
		if d.upload != nil || d.savePending || d.uploadSerial == ^uint32(0) {
			return failure(request.ID, "TOO_MANY_REQUESTS", "A text save is already pending.")
		}
		d.uploadSerial++
		d.upload = &saveUpload{token: d.uploadSerial, name: *p.Name, expected: *p.Bytes, text: make([]byte, 0, *p.Bytes)}
		return Response{Version: Version, ID: request.ID, OK: true, Result: struct {
			Token uint32 `json:"token"`
		}{d.uploadSerial}}
	case "file.appendSave":
		var p struct {
			Token  uint32  `json:"token"`
			Offset *int    `json:"offset"`
			Text   *string `json:"text"`
		}
		if decodeSaveParams(request.Params, &p) != nil || p.Offset == nil || p.Text == nil || d.upload == nil || p.Token != d.upload.token || *p.Offset != len(d.upload.text) || fileopen.ValidateSaveText(*p.Text) != nil || len(*p.Text) > d.upload.expected-len(d.upload.text) {
			return invalid()
		}
		d.upload.text = append(d.upload.text, (*p.Text)...)
		return Response{Version: Version, ID: request.ID, OK: true, Result: struct {
			Bytes int `json:"bytes"`
		}{len(d.upload.text)}}
	case "file.cancelSave":
		var p struct {
			Token uint32 `json:"token"`
		}
		if decodeSaveParams(request.Params, &p) != nil || d.upload == nil || p.Token != d.upload.token {
			return invalid()
		}
		d.upload = nil
		return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
	default:
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "Saving requires asynchronous dispatch.")
	}
}

func (d *Dispatcher) commitSave(request Request, finish func(Response)) {
	if _, granted := d.permissions[PermissionFileSave]; !granted {
		finish(failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted."))
		return
	}
	var p struct {
		Token uint32 `json:"token"`
	}
	if decodeSaveParams(request.Params, &p) != nil {
		finish(failure(request.ID, "INVALID_PARAMS", "The save parameters are invalid."))
		return
	}
	d.mu.Lock()
	if d.closing {
		d.mu.Unlock()
		finish(failure(request.ID, "SHUTTING_DOWN", "The native host is shutting down."))
		return
	}
	upload, saver := d.upload, d.saver
	if upload == nil || p.Token != upload.token || len(upload.text) != upload.expected || saver == nil || d.savePending {
		d.mu.Unlock()
		finish(failure(request.ID, "INVALID_PARAMS", "The prepared text is missing or incomplete."))
		return
	}
	d.upload = nil
	d.savePending = true
	d.mu.Unlock()
	var once sync.Once
	respond := func(result fileopen.SaveResult, err error) {
		once.Do(func() {
			d.mu.Lock()
			d.savePending = false
			d.mu.Unlock()
			if err != nil {
				code, message := "NATIVE_OPERATION_FAILED", "The selected file could not be saved."
				switch {
				case errors.Is(err, fileopen.ErrBusy):
					code, message = "TOO_MANY_REQUESTS", fileopen.ErrBusy.Error()
				case errors.Is(err, fileopen.ErrTooLarge):
					code, message = "PAYLOAD_TOO_LARGE", fileopen.ErrTooLarge.Error()
				case errors.Is(err, fileopen.ErrUnsupported):
					code, message = "UNSUPPORTED_FILE", fileopen.ErrUnsupported.Error()
				case errors.Is(err, fileopen.ErrRecovery):
					code, message = "SAVE_RECOVERY_REQUIRED", fileopen.ErrRecovery.Error()
				}
				finish(failure(request.ID, code, message))
				return
			}
			finish(Response{Version: Version, ID: request.ID, OK: true, Result: result})
		})
	}
	if err := saver.Save(string(upload.text), upload.name, respond); err != nil {
		respond(fileopen.SaveResult{}, err)
	}
}
