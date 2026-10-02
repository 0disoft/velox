package ipc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/0disoft/velox/internal/externalurl"
)

type fakeExternal struct {
	calls  int
	target string
	err    error
}

func (f *fakeExternal) Open(target string) error { f.calls++; f.target = target; return f.err }

func TestExternalDispatchPermissionValidationAndQueue(t *testing.T) {
	fake := &fakeExternal{}
	denied := NewDispatcher(Identity{}, nil, &fakeWindow{})
	denied.SetExternalOpener(fake)
	if response := denied.Dispatch(request(1, "external.open", `{"url":"https://example.com"}`)); response.Error == nil || response.Error.Code != "PERMISSION_DENIED" || fake.calls != 0 {
		t.Fatal("permission bypass")
	}
	dispatcher := NewDispatcher(Identity{}, []string{PermissionExternal}, &fakeWindow{})
	dispatcher.SetExternalOpener(fake)
	for _, params := range []string{`{}`, `{"url":null}`, `{"url":false}`, `{"URL":"https://example.com"}`, `{"url":"file:///C:/x"}`, `{"url":"https://example.com","extra":true}`, `{"url":"https://example.com","url":"https://other.com"}`} {
		response := dispatcher.Dispatch(request(2, "external.open", params))
		if response.OK || fake.calls != 0 {
			t.Fatalf("invalid dispatch=%+v", response)
		}
	}
	response := dispatcher.Dispatch(request(3, "external.open", `{"url":"HTTPS://EXAMPLE.COM/path"}`))
	body, _ := json.Marshal(response.Result)
	if !response.OK || string(body) != `{"queued":true}` || fake.target != "https://example.com/path" || fake.calls != 1 {
		t.Fatalf("response=%+v, target=%q", response, fake.target)
	}
	fake.err = externalurl.ErrBusy
	if response := dispatcher.Dispatch(request(4, "external.open", `{"url":"https://example.com"}`)); response.Error == nil || response.Error.Code != "TOO_MANY_REQUESTS" {
		t.Fatal("busy result incorrect")
	}
	fake.err = errors.New("private URL and native details")
	if response := dispatcher.Dispatch(request(5, "external.open", `{"url":"https://example.com"}`)); response.Error == nil || response.Error.Message != "The native operation failed." {
		t.Fatal("native details exposed")
	}
	dispatcher.Close()
	if response := dispatcher.Dispatch(request(6, "external.open", `{"url":"https://example.com"}`)); response.Error == nil || response.Error.Code != "SHUTTING_DOWN" {
		t.Fatal("shutdown accepted request")
	}
}
