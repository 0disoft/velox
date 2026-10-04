package ipc

import "encoding/json"

func (d *Dispatcher) windowProgress(request Request) Response {
	invalid := func() Response {
		return failure(request.ID, "INVALID_PARAMS", "Progress requires a state and, for normal, error or paused, an integer value from 0 to 100.")
	}
	var params map[string]json.RawMessage
	if json.Unmarshal(request.Params, &params) != nil {
		return invalid()
	}
	var state string
	if json.Unmarshal(params["state"], &state) != nil {
		return invalid()
	}
	var value uint32
	switch state {
	case "none", "indeterminate":
		if len(params) != 1 {
			return invalid()
		}
	case "normal", "error", "paused":
		if len(params) != 2 || params["value"] == nil || string(params["value"]) == "null" || json.Unmarshal(params["value"], &value) != nil || value > 100 {
			return invalid()
		}
	default:
		return invalid()
	}
	if d.window == nil || d.window.SetProgress(state, value) != nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
}
