package ipc

import (
	"testing"

	"github.com/0disoft/velox/internal/fileopen"
)

type fakeFolderAccess struct {
	calls, clears int
	released      uint32
	done          func(fileopen.FolderResult, error)
	listed        uint32
	listDone      func(fileopen.FolderListing, error)
	readTarget    uint32
	readName      string
	readDone      func(fileopen.Result, error)
	err           error
}

func (f *fakeFolderAccess) Select(done func(fileopen.FolderResult, error)) error {
	f.calls++
	f.done = done
	return f.err
}

func (f *fakeFolderAccess) List(target uint32, done func(fileopen.FolderListing, error)) error {
	f.listed, f.listDone = target, done
	return f.err
}
func (f *fakeFolderAccess) OpenText(target uint32, name string, done func(fileopen.Result, error)) error {
	f.readTarget, f.readName, f.readDone = target, name, done
	return f.err
}
func (f *fakeFolderAccess) ReleaseTarget(target uint32) error { f.released = target; return f.err }
func (f *fakeFolderAccess) ClearTarget()                      { f.clears++ }

func TestFolderSelectionPermissionParamsAndDeferredDispatch(t *testing.T) {
	for _, tc := range []struct {
		permissions  []string
		params, code string
	}{
		{nil, `{}`, "PERMISSION_DENIED"},
		{[]string{PermissionFileOpen}, `{}`, "PERMISSION_DENIED"},
		{[]string{PermissionFolderRead}, `{"path":"private"}`, "INVALID_PARAMS"},
	} {
		d := NewDispatcher(Identity{}, tc.permissions, &fakeWindow{})
		f := &fakeFolderAccess{}
		d.SetFolderAccess(f)
		d.DispatchAsync(request(1, "folder.select", tc.params), func(r Response) {
			if r.Error == nil || r.Error.Code != tc.code {
				t.Fatal(r)
			}
		})
		if f.calls != 0 {
			t.Fatal("unauthorized selection")
		}
	}
	d := NewDispatcher(Identity{}, []string{PermissionFolderRead}, &fakeWindow{})
	f := &fakeFolderAccess{}
	d.SetFolderAccess(f)
	raw := request(1, "folder.select", `{}`)
	if !RequiresDeferred(raw) {
		t.Fatal("not deferred")
	}
	responses := 0
	d.DispatchAsync(raw, func(r Response) {
		responses++
		if !r.OK {
			t.Fatal(r)
		}
	})
	if r := d.Dispatch(raw); r.Error == nil || r.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatal(r)
	}
	f.done(fileopen.FolderResult{Cancelled: true}, nil)
	f.done(fileopen.FolderResult{}, nil)
	if responses != 1 {
		t.Fatal("duplicate completion")
	}
	if r := d.Dispatch(request(2, "folder.select", `{}`)); r.Error == nil || r.Error.Code != "NATIVE_OPERATION_FAILED" {
		t.Fatal("sync dialog allowed", r)
	}
}

func TestFolderReleaseStrictTokensAndLifecycleCleanup(t *testing.T) {
	d := NewDispatcher(Identity{}, []string{PermissionFolderRead}, &fakeWindow{})
	f := &fakeFolderAccess{}
	d.SetFolderAccess(f)
	for _, params := range []string{`{}`, `{"target":0}`, `{"target":null}`, `{"target":-1}`, `{"target":1.5}`, `{"target":4294967296}`, `{"target":1,"path":"private"}`} {
		r := d.Dispatch(request(1, "folder.release", params))
		if r.Error == nil || r.Error.Code != "INVALID_PARAMS" {
			t.Fatal("bad token accepted", params, r)
		}
	}
	if r := d.Dispatch(request(1, "folder.release", `{"target":7}`)); !r.OK || f.released != 7 {
		t.Fatal(r)
	}
	d.DropFolderTarget()
	d.Close()
	if f.clears != 2 {
		t.Fatal("target not cleared")
	}
}

func TestFolderErrorsAndCloseWhileSelectionPending(t *testing.T) {
	for err, code := range map[error]string{fileopen.ErrBusy: "TOO_MANY_REQUESTS", fileopen.ErrFolderUnsupported: "UNSUPPORTED_FOLDER", fileopen.ErrFolderTarget: "FOLDER_TARGET_INVALID"} {
		d := NewDispatcher(Identity{}, []string{PermissionFolderRead}, &fakeWindow{})
		d.SetFolderAccess(&fakeFolderAccess{err: err})
		d.DispatchAsync(request(1, "folder.select", `{}`), func(r Response) {
			if r.Error == nil || r.Error.Code != code {
				t.Fatal(r)
			}
		})
	}
	d := NewDispatcher(Identity{}, []string{PermissionFolderRead}, &fakeWindow{})
	f := &fakeFolderAccess{}
	d.SetFolderAccess(f)
	d.DispatchAsync(request(1, "folder.select", `{}`), func(r Response) {
		if r.Error == nil || r.Error.Code != "SHUTTING_DOWN" {
			t.Fatal(r)
		}
	})
	d.Close()
	f.done(fileopen.FolderResult{}, nil)
}

func TestFolderListPermissionTokensDeferredCompletionAndRevocation(t *testing.T) {
	for _, tc := range []struct {
		permissions  []string
		params, code string
	}{
		{nil, `{"target":1}`, "PERMISSION_DENIED"},
		{[]string{PermissionFileOpen}, `{"target":1}`, "PERMISSION_DENIED"},
		{[]string{PermissionFolderRead}, `{"target":1,"path":"private"}`, "INVALID_PARAMS"},
		{[]string{PermissionFolderRead}, `{"target":1,"recursive":true}`, "INVALID_PARAMS"},
		{[]string{PermissionFolderRead}, `{"target":0}`, "INVALID_PARAMS"},
		{[]string{PermissionFolderRead}, `{"target":null}`, "INVALID_PARAMS"},
	} {
		d := NewDispatcher(Identity{}, tc.permissions, &fakeWindow{})
		f := &fakeFolderAccess{}
		d.SetFolderAccess(f)
		d.DispatchAsync(request(1, "folder.list", tc.params), func(r Response) {
			if r.Error == nil || r.Error.Code != tc.code {
				t.Fatal(r)
			}
		})
		if f.listed != 0 {
			t.Fatal("invalid request listed folder")
		}
	}
	d := NewDispatcher(Identity{}, []string{PermissionFolderRead}, &fakeWindow{})
	f := &fakeFolderAccess{}
	d.SetFolderAccess(f)
	raw := request(1, "folder.list", `{"target":7}`)
	if !RequiresDeferred(raw) {
		t.Fatal("list not deferred")
	}
	responses := 0
	d.DispatchAsync(raw, func(r Response) {
		responses++
		if !r.OK {
			t.Fatal(r)
		}
	})
	if f.listed != 7 {
		t.Fatal("wrong target")
	}
	if r := d.Dispatch(raw); r.Error == nil || r.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatal(r)
	}
	f.listDone(fileopen.FolderListing{Entries: []fileopen.FolderEntry{}}, nil)
	f.listDone(fileopen.FolderListing{}, nil)
	if responses != 1 {
		t.Fatal("duplicate completion")
	}
	d.DispatchAsync(request(2, "folder.list", `{"target":7}`), func(r Response) {
		if r.Error == nil || r.Error.Code != "FOLDER_TARGET_INVALID" {
			t.Fatal(r)
		}
	})
	f.listDone(fileopen.FolderListing{}, fileopen.ErrFolderTarget)
}
