package ipc

import (
	"errors"
	"testing"

	"github.com/0disoft/velox/internal/fileopen"
)

func TestFolderTextRequiresBothPermissionsAndStrictBasenameParams(t *testing.T) {
	permissions := []string{PermissionFolderRead, PermissionFolderReadText}
	for _, granted := range [][]string{nil, {PermissionFileOpen}, {PermissionFolderRead}, {PermissionFolderReadText}, {PermissionFolderRead, PermissionFileOpen}} {
		d := NewDispatcher(Identity{}, granted, &fakeWindow{})
		f := &fakeFolderAccess{}
		d.SetFolderAccess(f)
		d.DispatchAsync(request(1, "folder.openText", `{"target":1,"name":"notes.txt"}`), func(r Response) {
			if r.Error == nil || r.Error.Code != "PERMISSION_DENIED" {
				t.Fatal(r)
			}
		})
		if f.readDone != nil {
			t.Fatal("content permission bypassed")
		}
	}
	for _, params := range []string{`{}`, `{"target":0,"name":"notes.txt"}`, `{"target":null,"name":"notes.txt"}`, `{"target":1}`, `{"target":1,"name":null}`, `{"target":1,"name":2}`, `{"target":1,"name":"../private"}`, `{"target":1,"name":"notes.txt:ads"}`, `{"target":1,"name":"CON"}`, `{"target":1,"name":"notes.txt","path":"private"}`, `{"target":1,"target":2,"name":"notes.txt"}`} {
		d := NewDispatcher(Identity{}, permissions, &fakeWindow{})
		f := &fakeFolderAccess{}
		d.SetFolderAccess(f)
		d.DispatchAsync(request(1, "folder.openText", params), func(r Response) {
			code := "INVALID_PARAMS"
			if params == `{"target":1,"target":2,"name":"notes.txt"}` {
				code = "INVALID_REQUEST"
			}
			if r.Error == nil || r.Error.Code != code {
				t.Fatal(params, r)
			}
		})
		if f.readDone != nil {
			t.Fatal("invalid request read file")
		}
	}
}

func TestFolderTextDeferredCompletionAndRedactedFailures(t *testing.T) {
	permissions := []string{PermissionFolderRead, PermissionFolderReadText}
	d := NewDispatcher(Identity{}, permissions, &fakeWindow{})
	f := &fakeFolderAccess{}
	d.SetFolderAccess(f)
	raw := request(1, "folder.openText", `{"target":7,"name":"notes.txt"}`)
	if !RequiresDeferred(raw) {
		t.Fatal("read not deferred")
	}
	responses := 0
	d.DispatchAsync(raw, func(r Response) {
		responses++
		if !r.OK || r.Result.(fileopen.Result).Text != "hello" {
			t.Fatal(r)
		}
	})
	if f.readTarget != 7 || f.readName != "notes.txt" {
		t.Fatal("wrong read target")
	}
	if r := d.Dispatch(raw); r.Error == nil || r.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatal(r)
	}
	f.readDone(fileopen.Result{Name: "notes.txt", Text: "hello", Bytes: 5}, nil)
	f.readDone(fileopen.Result{}, nil)
	if responses != 1 {
		t.Fatal("duplicate completion")
	}
	for err, code := range map[error]string{fileopen.ErrTooLarge: "PAYLOAD_TOO_LARGE", fileopen.ErrUnsupported: "UNSUPPORTED_FILE", fileopen.ErrFolderTarget: "FOLDER_TARGET_INVALID", fileopen.ErrBusy: "TOO_MANY_REQUESTS", errors.New(`C:\private\notes.txt`): "NATIVE_OPERATION_FAILED"} {
		f.err = err
		d.DispatchAsync(request(2, "folder.openText", `{"target":7,"name":"notes.txt"}`), func(r Response) {
			if r.Error == nil || r.Error.Code != code || r.Result != nil {
				t.Fatal(r)
			}
			if r.Error.Message == err.Error() && code == "NATIVE_OPERATION_FAILED" {
				t.Fatal("private error leaked")
			}
		})
	}
}

func TestFolderTextCloseDropsPendingResponse(t *testing.T) {
	d := NewDispatcher(Identity{}, []string{PermissionFolderRead, PermissionFolderReadText}, &fakeWindow{})
	f := &fakeFolderAccess{}
	d.SetFolderAccess(f)
	d.DispatchAsync(request(1, "folder.openText", `{"target":7,"name":"notes.txt"}`), func(r Response) {
		if r.Error == nil || r.Error.Code != "SHUTTING_DOWN" || r.Result != nil {
			t.Fatal("private response survived close", r)
		}
	})
	d.Close()
	f.readDone(fileopen.Result{Text: "private"}, nil)
}
