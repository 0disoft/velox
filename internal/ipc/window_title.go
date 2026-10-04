package ipc

import (
	"encoding/json"
	"unicode"
)

const MaxWindowTitleBytes = 512

func (d *Dispatcher) setWindowTitle(request Request) Response {
	var params map[string]json.RawMessage
	if json.Unmarshal(request.Params, &params) != nil || len(params) != 1 || params["title"] == nil {
		return failure(request.ID, "INVALID_PARAMS", "Window title parameters must contain only a title string.")
	}
	var title string
	if string(params["title"]) == "null" || json.Unmarshal(params["title"], &title) != nil {
		return failure(request.ID, "INVALID_PARAMS", "Window title parameters must contain only a title string.")
	}
	if len(title) > MaxWindowTitleBytes {
		return failure(request.ID, "PAYLOAD_TOO_LARGE", "Window title exceeds 512 UTF-8 bytes.")
	}
	for _, char := range title {
		if unicode.IsControl(char) {
			return failure(request.ID, "INVALID_PARAMS", "Window title must not contain control characters.")
		}
	}
	if title == "" {
		title = d.identity.Name
	}
	if d.window == nil || d.window.SetTitle(title) != nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
}
