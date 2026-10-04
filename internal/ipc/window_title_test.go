package ipc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestWindowTitlePermissionAndIdentity(t *testing.T) {
	window := &fakeWindow{}
	identity := Identity{Name: "Original app"}
	for _, permissions := range [][]string{nil, {PermissionWindow}} {
		d := NewDispatcher(identity, permissions, window)
		response := d.Dispatch(request(1, "window.setTitle", `{"title":"Changed"}`))
		if response.Error == nil || response.Error.Code != "PERMISSION_DENIED" || window.title != "" {
			t.Fatalf("denied title update = %+v, title %q", response, window.title)
		}
	}
	d := NewDispatcher(identity, []string{PermissionWindowTitle, PermissionAppInfo}, window)
	for _, title := range []string{"\uD55C\uAE00.md - App", strings.Repeat("a", MaxWindowTitleBytes), strings.Repeat("\uAC00", 170), ""} {
		params, _ := json.Marshal(map[string]string{"title": title})
		if response := d.Dispatch(request(1, "window.setTitle", string(params))); !response.OK {
			t.Fatalf("title update = %+v", response)
		}
		want := title
		if want == "" {
			want = identity.Name
		}
		if window.title != want {
			t.Fatalf("title = %q, want %q", window.title, want)
		}
	}
	info := d.Dispatch(request(2, "app.getInfo", `{}`))
	if info.Result != identity {
		t.Fatalf("title changed identity: %+v", info)
	}
	if response := d.Dispatch(request(3, "window.minimize", `{}`)); response.Error == nil || response.Error.Code != "PERMISSION_DENIED" {
		t.Fatalf("title grant also allowed window.basic: %+v", response)
	}
	if RequiresDeferred(request(4, "window.setTitle", `{"title":"test"}`)) {
		t.Fatal("window title unexpectedly deferred")
	}
	d.Close()
	if response := d.Dispatch(request(5, "window.setTitle", `{"title":"late"}`)); response.Error == nil || response.Error.Code != "SHUTTING_DOWN" {
		t.Fatalf("shutdown update = %+v", response)
	}
}

func TestWindowTitleRejectsInvalidParams(t *testing.T) {
	window := &fakeWindow{}
	d := NewDispatcher(Identity{}, []string{PermissionWindowTitle}, window)
	for _, params := range []string{`{}`, `{"title":null}`, `{"title":1}`, `{"title":true}`,
		`{"title":[]}`, `{"title":"ok","extra":true}`, `{"title":"a\n"}`, `{"title":"a\t"}`,
		`{"title":"a\u0000"}`, `{"title":"a\u007f"}`, `{"title":"a\u0085"}`} {
		response := d.Dispatch(request(1, "window.setTitle", params))
		if response.Error == nil || response.Error.Code != "INVALID_PARAMS" || window.title != "" {
			t.Fatalf("invalid %s = %+v", params, response)
		}
	}
	for _, title := range []string{strings.Repeat("a", 513), strings.Repeat("\uAC00", 171)} {
		params, _ := json.Marshal(map[string]string{"title": title})
		response := d.Dispatch(request(1, "window.setTitle", string(params)))
		if response.Error == nil || response.Error.Code != "PAYLOAD_TOO_LARGE" {
			t.Fatalf("oversized title = %+v", response)
		}
	}
}

func TestWindowTitleRedactsNativeFailure(t *testing.T) {
	for _, window := range []Window{nil, &fakeWindow{operationErr: errors.New("private native details")}} {
		d := NewDispatcher(Identity{}, []string{PermissionWindowTitle}, window)
		response := d.Dispatch(request(1, "window.setTitle", `{"title":"test"}`))
		if response.Error == nil || response.Error.Code != "NATIVE_OPERATION_FAILED" || strings.Contains(response.Error.Message, "private") {
			t.Fatalf("native failure = %+v", response)
		}
	}
}
