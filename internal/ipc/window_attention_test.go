package ipc

import (
	"errors"
	"strings"
	"testing"
)

func TestWindowAttentionIndependentPermission(t *testing.T) {
	for _, permissions := range [][]string{nil, {PermissionWindow}, {PermissionWindowTitle}} {
		window := &fakeWindow{}
		d := NewDispatcher(Identity{}, permissions, window)
		for _, method := range []string{"window.requestAttention", "window.cancelAttention"} {
			response := d.Dispatch(request(1, method, `{}`))
			if response.Error == nil || response.Error.Code != "PERMISSION_DENIED" || window.attentionCount != 0 || window.attentionCancelled != 0 {
				t.Fatalf("denied %s = %+v", method, response)
			}
		}
	}
	d := NewDispatcher(Identity{}, []string{PermissionWindowAttention}, &fakeWindow{})
	for _, method := range []string{"window.minimize", "window.setTitle"} {
		if response := d.Dispatch(request(1, method, `{}`)); response.Error == nil || response.Error.Code != "PERMISSION_DENIED" {
			t.Fatalf("attention grants unrelated %s: %+v", method, response)
		}
	}
}

func TestWindowAttentionCountAndCancel(t *testing.T) {
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionWindowAttention}, window)
	for _, test := range []struct {
		params string
		count  uint32
	}{
		{`{}`, 3}, {`{"count":1}`, 1}, {`{"count":5}`, 5},
	} {
		raw := request(1, "window.requestAttention", test.params)
		if RequiresDeferred(raw) {
			t.Fatal("attention unexpectedly deferred")
		}
		response := d.Dispatch(raw)
		if !response.OK || window.attentionCount != test.count {
			t.Fatalf("attention %s = %+v, count %d", test.params, response, window.attentionCount)
		}
	}
	if response := d.Dispatch(request(2, "window.cancelAttention", `{}`)); !response.OK || window.attentionCancelled != 1 {
		t.Fatalf("cancel = %+v", response)
	}
	d.Close()
	for _, method := range []string{"window.requestAttention", "window.cancelAttention"} {
		response := d.Dispatch(request(3, method, `{}`))
		if response.Error == nil || response.Error.Code != "SHUTTING_DOWN" {
			t.Fatalf("late %s = %+v", method, response)
		}
	}
}

func TestWindowAttentionRejectsInvalidParams(t *testing.T) {
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionWindowAttention}, window)
	for _, params := range []string{`{"count":0}`, `{"count":6}`, `{"count":-1}`, `{"count":1.5}`,
		`{"count":1.0}`, `{"count":4294967296}`, `{"count":null}`, `{"count":true}`,
		`{"count":"3"}`, `{"count":[]}`, `{"extra":1}`, `{"count":3,"extra":1}`} {
		response := d.Dispatch(request(1, "window.requestAttention", params))
		if response.Error == nil || response.Error.Code != "INVALID_PARAMS" || window.attentionCount != 0 {
			t.Fatalf("invalid %s = %+v", params, response)
		}
	}
	if response := d.Dispatch(request(1, "window.cancelAttention", `{"count":1}`)); response.Error == nil || response.Error.Code != "INVALID_PARAMS" || window.attentionCancelled != 0 {
		t.Fatalf("cancel params = %+v", response)
	}
}

func TestWindowAttentionRedactsNativeErrors(t *testing.T) {
	for _, window := range []Window{nil, &fakeWindow{operationErr: errors.New("private native details")}} {
		d := NewDispatcher(Identity{}, []string{PermissionWindowAttention}, window)
		for _, method := range []string{"window.requestAttention", "window.cancelAttention"} {
			response := d.Dispatch(request(1, method, `{}`))
			if response.Error == nil || response.Error.Code != "NATIVE_OPERATION_FAILED" || strings.Contains(response.Error.Message, "private") {
				t.Fatalf("native failure = %+v", response)
			}
		}
	}
}
