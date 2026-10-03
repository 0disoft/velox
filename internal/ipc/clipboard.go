package ipc

import (
	"encoding/json"
	"errors"

	"github.com/0disoft/velox/internal/clipboard"
)

const PermissionClipboardWrite = "clipboard.write"

type ClipboardWriter interface{ WriteText(string) error }

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
