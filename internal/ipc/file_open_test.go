package ipc

import (
	"errors"
	"testing"

	"github.com/0disoft/velox/internal/fileopen"
)

type fakeFileOpener struct {
	complete func(fileopen.Result, error)
	calls    int
	err      error
}

func (f *fakeFileOpener) Open(done func(fileopen.Result, error)) error {
	f.calls++
	f.complete = done
	return f.err
}

func TestFileOpenPermissionAndNoPathParameters(t *testing.T) {
	for _, tc := range []struct {
		permissions  []string
		params, code string
	}{
		{nil, `{}`, "PERMISSION_DENIED"},
		{[]string{PermissionFileOpen}, `{"path":"C:/private.txt"}`, "INVALID_PARAMS"},
	} {
		d := NewDispatcher(Identity{}, tc.permissions, &fakeWindow{})
		opener := &fakeFileOpener{}
		d.SetFileOpener(opener)
		d.DispatchAsync(request(1, "file.openText", tc.params), func(r Response) {
			if r.Error == nil || r.Error.Code != tc.code {
				t.Fatal(r)
			}
		})
		if opener.calls != 0 {
			t.Fatal("denied request reached picker")
		}
	}
}

func TestFileOpenRetainsIDAndCompletesOnce(t *testing.T) {
	d := NewDispatcher(Identity{}, []string{PermissionFileOpen}, &fakeWindow{})
	opener := &fakeFileOpener{}
	d.SetFileOpener(opener)
	responses := 0
	d.DispatchAsync(request(1, "file.openText", `{}`), func(r Response) {
		responses++
		if !r.OK {
			t.Fatal(r)
		}
	})
	if r := d.Dispatch(request(1, "file.openText", `{}`)); r.Error == nil || r.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatal(r)
	}
	opener.complete(fileopen.Result{Cancelled: true}, nil)
	opener.complete(fileopen.Result{}, errors.New("duplicate"))
	if responses != 1 {
		t.Fatal("completed more than once")
	}
	if r := d.Dispatch(request(1, "app.getInfo", `{}`)); r.Error != nil && r.Error.Code == "DUPLICATE_REQUEST_ID" {
		t.Fatal("ID was not released")
	}
}

func TestFileOpenErrorsAreStableAndPrivate(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{fileopen.ErrBusy, "TOO_MANY_REQUESTS"}, {fileopen.ErrTooLarge, "PAYLOAD_TOO_LARGE"}, {fileopen.ErrUnsupported, "UNSUPPORTED_FILE"},
		{errors.New("C:/secret/file.txt: private failure"), "NATIVE_OPERATION_FAILED"},
	} {
		d := NewDispatcher(Identity{}, []string{PermissionFileOpen}, &fakeWindow{})
		d.SetFileOpener(&fakeFileOpener{err: tc.err})
		called := false
		d.DispatchAsync(request(1, "file.openText", `{}`), func(r Response) {
			called = true
			if r.Error == nil || r.Error.Code != tc.code || r.Error.Message == "C:/secret/file.txt: private failure" {
				t.Fatal(r)
			}
		})
		if !called {
			t.Fatal("admission error did not complete")
		}
	}
	closed := NewDispatcher(Identity{}, []string{PermissionFileOpen}, &fakeWindow{})
	opener := &fakeFileOpener{}
	closed.SetFileOpener(opener)
	closed.DispatchAsync(request(1, "file.openText", `{}`), func(r Response) {
		if r.Error == nil || r.Error.Code != "SHUTTING_DOWN" {
			t.Fatal(r)
		}
	})
	closed.Close()
	opener.complete(fileopen.Result{Text: "private"}, nil)
}
