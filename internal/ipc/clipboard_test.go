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
	if r := unavailable.Dispatch(request(9, "clipboard.readText", `{}`)); r.Error == nil || r.Error.Code != "PERMISSION_DENIED" {
		t.Fatal("clipboard write grants reading")
	}
}

type fakeClipboardReader struct {
	done  func(clipboard.ReadResult, error)
	calls int
	err   error
}

func (f *fakeClipboardReader) ReadText(done func(clipboard.ReadResult, error)) error {
	f.calls++
	f.done = done
	return f.err
}

func TestClipboardReadAsyncPermissionAndCompletion(t *testing.T) {
	fake := &fakeClipboardReader{}
	d := NewDispatcher(Identity{}, []string{PermissionClipboardRead}, &fakeWindow{})
	d.SetClipboardReader(fake)
	var response Response
	replies := 0
	reply := func(r Response) { response = r; replies++ }
	raw := request(10, "clipboard.readText", `{}`)
	if !RequiresDeferred(raw) {
		t.Fatal("read confirmation is not deferred")
	}
	if r := d.Dispatch(raw); r.OK || fake.calls != 0 {
		t.Fatal("synchronous confirmation allowed")
	}
	d.DispatchAsync(request(11, "clipboard.readText", `{"text":"unexpected"}`), reply)
	if response.Error == nil || response.Error.Code != "INVALID_PARAMS" || fake.calls != 0 {
		t.Fatal(response)
	}
	denied := NewDispatcher(Identity{}, []string{PermissionClipboardWrite}, &fakeWindow{})
	denied.SetClipboardReader(fake)
	denied.DispatchAsync(raw, reply)
	if response.Error == nil || response.Error.Code != "PERMISSION_DENIED" || fake.calls != 0 {
		t.Fatal(response)
	}
	replies = 0
	d.DispatchAsync(raw, reply)
	if replies != 0 || fake.calls != 1 {
		t.Fatal("read completed before approval")
	}
	d.DispatchAsync(raw, reply)
	if response.Error == nil || response.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatal(response)
	}
	text := "\ud55c\uae00\n\U0001f642"
	fake.done(clipboard.ReadResult{Text: &text}, nil)
	if !response.OK || replies != 2 {
		t.Fatal(response)
	}
	fake.done(clipboard.ReadResult{Text: &text}, nil)
	if replies != 2 {
		t.Fatal("duplicate completion")
	}
	d.DispatchAsync(raw, reply)
	fake.done(clipboard.ReadResult{Cancelled: true, Text: &text}, nil)
	body, _ := json.Marshal(response.Result)
	if !response.OK || string(body) != `{"cancelled":true}` {
		t.Fatal("cancellation leaked text", string(body))
	}
	d.DispatchAsync(raw, reply)
	d.Close()
	fake.done(clipboard.ReadResult{Text: &text}, nil)
	if response.Error == nil || response.Error.Code != "SHUTTING_DOWN" || response.Result != nil {
		t.Fatal("late read leaked", response)
	}
}

func TestClipboardReadErrorsAndResultValidation(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{clipboard.ErrPending, "TOO_MANY_REQUESTS"}, {clipboard.ErrBusy, "CLIPBOARD_BUSY"},
		{clipboard.ErrUnsupported, "UNSUPPORTED_TEXT"}, {clipboard.ErrInvalidText, "UNSUPPORTED_TEXT"},
		{clipboard.ErrTooLarge, "PAYLOAD_TOO_LARGE"}, {errors.New("private contents"), "NATIVE_OPERATION_FAILED"},
	} {
		d := NewDispatcher(Identity{}, []string{PermissionClipboardRead}, &fakeWindow{})
		fake := &fakeClipboardReader{err: test.err}
		d.SetClipboardReader(fake)
		d.DispatchAsync(request(1, "clipboard.readText", `{}`), func(r Response) {
			if r.Error == nil || r.Error.Code != test.code || strings.Contains(r.Error.Message, "private") || r.Result != nil {
				t.Fatal(r)
			}
		})
	}
	for _, text := range []string{"", "\ud55c\uae00", strings.Repeat("x", clipboard.MaxTextBytes+1), "a\x00b"} {
		d := NewDispatcher(Identity{}, []string{PermissionClipboardRead}, &fakeWindow{})
		fake := &fakeClipboardReader{}
		d.SetClipboardReader(fake)
		d.DispatchAsync(request(1, "clipboard.readText", `{}`), func(r Response) {
			if r.OK != (clipboard.Validate(text) == nil) {
				t.Fatal("invalid read result", r)
			}
		})
		fake.done(clipboard.ReadResult{Text: &text}, nil)
	}
}
