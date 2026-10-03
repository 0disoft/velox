package ipc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/clipboard"
)

type fakeClipboard struct {
	calls int
	text  string
	err   error
}

func (f *fakeClipboard) WriteText(text string) error { f.calls++; f.text = text; return f.err }

func TestClipboardPermissionParamsAndShutdown(t *testing.T) {
	fake := &fakeClipboard{}
	denied := NewDispatcher(Identity{}, nil, &fakeWindow{})
	denied.SetClipboardWriter(fake)
	if r := denied.Dispatch(request(1, "clipboard.writeText", `{"text":"hello"}`)); r.Error == nil || r.Error.Code != "PERMISSION_DENIED" || fake.calls != 0 {
		t.Fatal("clipboard permission bypass")
	}
	d := NewDispatcher(Identity{}, []string{PermissionClipboardWrite}, &fakeWindow{})
	d.SetClipboardWriter(fake)
	for _, params := range []string{`{}`, `{"text":null}`, `{"text":true}`, `{"text":1}`, `{"text":{}}`, `{"text":"x","extra":true}`, `{"text":"x","text":"y"}`, `{"text":"a\u0000b"}`} {
		if r := d.Dispatch(request(2, "clipboard.writeText", params)); r.OK || fake.calls != 0 {
			t.Fatalf("invalid clipboard request accepted: %+v", r)
		}
	}
	for _, text := range []string{"", "hello\n\ud55c\uae00 \U0001f642", strings.Repeat("x", clipboard.MaxTextBytes)} {
		params, _ := json.Marshal(map[string]string{"text": text})
		r := d.Dispatch(request(3, "clipboard.writeText", string(params)))
		body, _ := json.Marshal(r.Result)
		if !r.OK || string(body) != "null" || fake.text != text {
			t.Fatalf("clipboard text not preserved: %+v", r)
		}
	}
	params, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", clipboard.MaxTextBytes+1)})
	if r := d.Dispatch(request(4, "clipboard.writeText", string(params))); r.Error == nil || r.Error.Code != "PAYLOAD_TOO_LARGE" || fake.calls != 3 {
		t.Fatal("clipboard byte budget not enforced")
	}
	fake.err = clipboard.ErrBusy
	if r := d.Dispatch(request(5, "clipboard.writeText", `{"text":"x"}`)); r.Error == nil || r.Error.Code != "CLIPBOARD_BUSY" {
		t.Fatal("clipboard busy error incorrect")
	}
	fake.err = errors.New("private text and native details")
	if r := d.Dispatch(request(6, "clipboard.writeText", `{"text":"x"}`)); r.Error == nil || r.Error.Message != "The clipboard operation failed." {
		t.Fatal("clipboard error leaked details")
	}
	d.Close()
	if r := d.Dispatch(request(7, "clipboard.writeText", `{"text":"x"}`)); r.Error == nil || r.Error.Code != "SHUTTING_DOWN" || fake.calls != 5 {
		t.Fatal("clipboard shutdown dispatch accepted")
	}
	unavailable := NewDispatcher(Identity{}, []string{PermissionClipboardWrite}, &fakeWindow{})
	if r := unavailable.Dispatch(request(8, "clipboard.writeText", `{"text":"x"}`)); r.Error == nil || r.Error.Code != "NATIVE_OPERATION_FAILED" {
		t.Fatal("unavailable clipboard writer accepted")
	}
	if r := unavailable.Dispatch(request(9, "clipboard.readText", `{}`)); r.Error == nil || r.Error.Code != "METHOD_NOT_FOUND" {
		t.Fatal("clipboard read method exists")
	}
}
