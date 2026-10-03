package ipc

import (
	"encoding/json"
	"errors"

	"github.com/0disoft/velox/internal/clipboard"
)

const PermissionClipboardWrite = "clipboard.write"
const PermissionClipboardRead = "clipboard.read"

type ClipboardWriter interface{ WriteText(string) error }

type ClipboardReader interface {
	ReadText(func(clipboard.ReadResult, error)) error
}

func (d *Dispatcher) SetClipboardReader(reader ClipboardReader) {
	d.mu.Lock()
	d.clipboardReader = reader
	d.mu.Unlock()
}

func (d *Dispatcher) readClipboard(request Request, finish func(Response)) {
	if _, granted := d.permissions[PermissionClipboardRead]; !granted {
		finish(failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted."))
		return
	}
	if err := requireEmptyParams(request.Params); err != nil {
		finish(failure(request.ID, "INVALID_PARAMS", err.Error()))
		return
	}
	d.mu.Lock()
	reader := d.clipboardReader
	d.mu.Unlock()
	if reader == nil {
		finish(failure(request.ID, "NATIVE_OPERATION_FAILED", "The clipboard operation failed."))
		return
	}
	respond := func(result clipboard.ReadResult, err error) {
		if err == nil && !result.Cancelled {
			if result.Text == nil {
				err = clipboard.ErrNative
			} else {
				err = clipboard.Validate(*result.Text)
			}
		}
		if err != nil {
			code, message := "NATIVE_OPERATION_FAILED", "The clipboard operation failed."
			switch {
			case errors.Is(err, clipboard.ErrPending):
				code, message = "TOO_MANY_REQUESTS", clipboard.ErrPending.Error()
			case errors.Is(err, clipboard.ErrBusy):
				code, message = "CLIPBOARD_BUSY", clipboard.ErrBusy.Error()
			case errors.Is(err, clipboard.ErrTooLarge):
				code, message = "PAYLOAD_TOO_LARGE", clipboard.ErrTooLarge.Error()
			case errors.Is(err, clipboard.ErrUnsupported), errors.Is(err, clipboard.ErrInvalidText):
				code, message = "UNSUPPORTED_TEXT", clipboard.ErrUnsupported.Error()
			}
			finish(failure(request.ID, code, message))
			return
		}
		if result.Cancelled {
			result.Text = nil
		}
		finish(Response{Version: Version, ID: request.ID, OK: true, Result: result})
	}
	if err := reader.ReadText(respond); err != nil {
		respond(clipboard.ReadResult{}, err)
	}
}

func (d *Dispatcher) SetClipboardWriter(writer ClipboardWriter) {
	d.mu.Lock()
	d.clipboard = writer
	d.mu.Unlock()
}

func (d *Dispatcher) writeClipboard(request Request) Response {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(request.Params, &params); err != nil || len(params) != 1 || params["text"] == nil {
		return failure(request.ID, "INVALID_PARAMS", "Clipboard parameters must contain only a text string.")
	}
	var text string
	if string(params["text"]) == "null" || json.Unmarshal(params["text"], &text) != nil {
		return failure(request.ID, "INVALID_PARAMS", "Clipboard parameters must contain only a text string.")
	}
	if err := clipboard.Validate(text); err != nil {
		code := "INVALID_PARAMS"
		if errors.Is(err, clipboard.ErrTooLarge) {
			code = "PAYLOAD_TOO_LARGE"
		}
		return failure(request.ID, code, err.Error())
	}
	d.mu.Lock()
	writer := d.clipboard
	d.mu.Unlock()
	if writer == nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The clipboard operation failed.")
	}
	if err := writer.WriteText(text); err != nil {
		if errors.Is(err, clipboard.ErrBusy) {
			return failure(request.ID, "CLIPBOARD_BUSY", clipboard.ErrBusy.Error())
		}
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The clipboard operation failed.")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
}
