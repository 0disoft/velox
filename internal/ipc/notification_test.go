package ipc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func notificationParams(kind, message string) string {
	data, _ := json.Marshal(map[string]string{"kind": kind, "message": message})
	return string(data)
}

func TestNotificationPermissionAndLifecycle(t *testing.T) {
	for _, permissions := range [][]string{nil, {PermissionWindow}, {PermissionWindowAttention}, {PermissionWindowProgress}} {
		window := &fakeWindow{}
		d := NewDispatcher(Identity{}, permissions, window)
		if r := d.Dispatch(request(1, "notification.show", notificationParams("info", "Ready"))); r.Error == nil || r.Error.Code != "PERMISSION_DENIED" || window.notificationText != "" {
			t.Fatal(r, window)
		}
	}
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionNotification}, window)
	for _, kind := range []string{"info", "warning", "error"} {
		raw := request(1, "notification.show", notificationParams(kind, "\uD55C\uAE00\n\tReady"))
		if RequiresDeferred(raw) {
			t.Fatal("notifications must stay on the UI thread")
		}
		r := d.Dispatch(raw)
		if !r.OK || string(r.Result.(json.RawMessage)) != "null" || window.notificationKind != kind || window.notificationText != "\uD55C\uAE00\n\tReady" {
			t.Fatal(r, window)
		}
	}
	if r := d.Dispatch(request(2, "window.minimize", `{}`)); r.Error == nil || r.Error.Code != "PERMISSION_DENIED" {
		t.Fatal(r)
	}
	d.Close()
	if r := d.Dispatch(request(3, "notification.show", notificationParams("info", "Ready"))); r.Error == nil || r.Error.Code != "SHUTTING_DOWN" {
		t.Fatal(r)
	}
}

func TestNotificationRejectsInvalidParams(t *testing.T) {
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionNotification}, window)
	for _, params := range []string{
		`{}`, `{"kind":"info"}`, `{"message":"Ready"}`, `{"kind":null,"message":"Ready"}`,
		`{"kind":1,"message":"Ready"}`, `{"kind":"info","message":null}`,
		`{"kind":"info","message":true}`, `{"kind":"info","message":"Ready","title":"Other app"}`,
		notificationParams("other", "Ready"), notificationParams("info", ""), notificationParams("info", " \n\t\u3000"),
		notificationParams("info", "a\x00b"), notificationParams("info", "a\rb"), notificationParams("info", "a\u0085b"),
		notificationParams("info", "a\x7fb"),
		notificationParams("info", strings.Repeat("a", 256)), notificationParams("info", strings.Repeat("\U0001F680", 128)),
	} {
		if r := d.Dispatch(request(1, "notification.show", params)); r.Error == nil || r.Error.Code != "INVALID_PARAMS" || window.notificationText != "" {
			t.Fatal(params, r)
		}
	}
	for _, message := range []string{strings.Repeat("a", 255), strings.Repeat("\uD55C", 255), strings.Repeat("\U0001F680", 127) + "a"} {
		if r := d.Dispatch(request(1, "notification.show", notificationParams("info", message))); !r.OK || window.notificationText != message {
			t.Fatal(r)
		}
	}
}

func TestNotificationRedactsNativeErrors(t *testing.T) {
	for _, window := range []Window{nil, &fakeWindow{operationErr: errors.New("private native path")}} {
		d := NewDispatcher(Identity{}, []string{PermissionNotification}, window)
		if r := d.Dispatch(request(1, "notification.show", notificationParams("info", "Ready"))); r.Error == nil || r.Error.Code != "NATIVE_OPERATION_FAILED" || strings.Contains(r.Error.Message, "private") {
			t.Fatal(r)
		}
	}
}
