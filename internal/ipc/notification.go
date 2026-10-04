package ipc

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf16"
)

// ValidNotification keeps the shell's fixed UTF-16 buffer from truncating text.
func ValidNotification(kind, message string) bool {
	if kind != "info" && kind != "warning" && kind != "error" {
		return false
	}
	if strings.TrimSpace(message) == "" || len(message) > 4*255 || len(utf16.Encode([]rune(message))) > 255 {
		return false
	}
	for _, r := range message {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}

func (d *Dispatcher) showNotification(request Request) Response {
	var params map[string]json.RawMessage
	var kind, message string
	if json.Unmarshal(request.Params, &params) != nil || len(params) != 2 ||
		json.Unmarshal(params["kind"], &kind) != nil || json.Unmarshal(params["message"], &message) != nil ||
		!ValidNotification(kind, message) {
		return failure(request.ID, "INVALID_PARAMS", "Notification requires info, warning or error and nonempty text of at most 255 UTF-16 units.")
	}
	if d.window == nil || d.window.ShowNotification(kind, message) != nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
}
