package ipc

import (
	"errors"
	"strings"
	"testing"
)

func TestWindowProgressPermissionAndLifecycle(t *testing.T) {
	for _, permissions := range [][]string{nil, {PermissionWindow}, {PermissionWindowTitle}, {PermissionWindowAttention}} {
		window := &fakeWindow{}
		d := NewDispatcher(Identity{}, permissions, window)
		response := d.Dispatch(request(1, "window.setProgress", `{"state":"normal","value":50}`))
		if response.Error == nil || response.Error.Code != "PERMISSION_DENIED" || window.progressState != "" {
			t.Fatal(response, window)
		}
	}
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionWindowProgress}, window)
	for _, method := range []string{"window.minimize", "window.setTitle", "window.requestAttention"} {
		if r := d.Dispatch(request(1, method, `{}`)); r.Error == nil || r.Error.Code != "PERMISSION_DENIED" {
			t.Fatal(r)
		}
	}
	for _, state := range []string{"none", "indeterminate", "normal", "error", "paused"} {
		params := `{"state":"` + state + `"}`
		if state != "none" && state != "indeterminate" {
			params = `{"state":"` + state + `","value":100}`
		}
		raw := request(1, "window.setProgress", params)
		if RequiresDeferred(raw) {
			t.Fatal("progress must stay on the UI thread")
		}
		if r := d.Dispatch(raw); !r.OK || window.progressState != state {
			t.Fatal(state, r)
		}
	}
	d.Close()
	if r := d.Dispatch(request(2, "window.setProgress", `{"state":"none"}`)); r.Error == nil || r.Error.Code != "SHUTTING_DOWN" {
		t.Fatal(r)
	}
}

func TestWindowProgressRejectsInvalidParams(t *testing.T) {
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionWindowProgress}, window)
	for _, params := range []string{
		`{}`, `{"state":null}`, `{"state":false}`, `{"state":"unknown"}`, `{"state":"normal"}`,
		`{"state":"none","value":0}`, `{"state":"indeterminate","value":1}`,
		`{"state":"normal","value":null}`, `{"state":"normal","value":-1}`, `{"state":"normal","value":101}`,
		`{"state":"normal","value":1.0}`, `{"state":"normal","value":1.5}`, `{"state":"normal","value":"50"}`,
		`{"state":"normal","value":true}`, `{"state":"normal","value":[]}`, `{"state":"normal","value":50,"extra":1}`,
	} {
		if r := d.Dispatch(request(1, "window.setProgress", params)); r.Error == nil || r.Error.Code != "INVALID_PARAMS" || window.progressState != "" {
			t.Fatal(params, r)
		}
	}
	if r := d.Dispatch(request(1, "window.setProgress", `{"state":"normal","value":0}`)); !r.OK || window.progressValue != 0 {
		t.Fatal(r)
	}
}

func TestWindowProgressRedactsNativeErrors(t *testing.T) {
	for _, window := range []Window{nil, &fakeWindow{operationErr: errors.New("private HRESULT")}} {
		d := NewDispatcher(Identity{}, []string{PermissionWindowProgress}, window)
		if r := d.Dispatch(request(1, "window.setProgress", `{"state":"none"}`)); r.Error == nil || r.Error.Code != "NATIVE_OPERATION_FAILED" || strings.Contains(r.Error.Message, "private") {
			t.Fatal(r)
		}
	}
}
