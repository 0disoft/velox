package ipc

import "encoding/json"

const MaxAttentionCount = 5

func (d *Dispatcher) windowAttention(request Request) Response {
	var count uint32 = 3
	if request.Method == "window.cancelAttention" {
		if err := requireEmptyParams(request.Params); err != nil {
			return failure(request.ID, "INVALID_PARAMS", err.Error())
		}
	} else {
		var params map[string]json.RawMessage
		if json.Unmarshal(request.Params, &params) != nil || len(params) > 1 {
			return failure(request.ID, "INVALID_PARAMS", "Attention parameters accept only an optional count integer from 1 to 5.")
		}
		if len(params) != 0 {
			raw := params["count"]
			if raw == nil || string(raw) == "null" || json.Unmarshal(raw, &count) != nil || count < 1 || count > MaxAttentionCount {
				return failure(request.ID, "INVALID_PARAMS", "Attention parameters accept only an optional count integer from 1 to 5.")
			}
		}
	}
	if d.window == nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	var err error
	if request.Method == "window.cancelAttention" {
		err = d.window.CancelAttention()
	} else {
		err = d.window.RequestAttention(count)
	}
	if err != nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: json.RawMessage("null")}
}
