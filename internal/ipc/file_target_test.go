package ipc

import (
	"errors"
	"testing"

	"github.com/0disoft/velox/internal/fileopen"
)

type fakeTargetSaver struct {
	fakeFileSaver
	selected int
	target   uint32
	cleared  int
}

func (f *fakeTargetSaver) SaveAs(text, name string, done func(fileopen.SaveResult, error)) error {
	f.selected++
	return f.Save(text, name, done)
}
func (f *fakeTargetSaver) SaveTo(text string, target uint32, done func(fileopen.SaveResult, error)) error {
	f.target = target
	return f.Save(text, "connected", done)
}
func (f *fakeTargetSaver) ReleaseTarget(target uint32) error {
	if target != 7 {
		return fileopen.ErrTarget
	}
	f.ClearTarget()
	return nil
}
func (f *fakeTargetSaver) ClearTarget() { f.cleared++ }

func TestConnectedSavePermissionAndParams(t *testing.T) {
	for _, method := range []string{"file.commitSaveAs", "file.commitSaveTo", "file.releaseSaveTarget"} {
		d := NewDispatcher(Identity{}, nil, &fakeWindow{})
		s := &fakeTargetSaver{}
		d.SetFileSaver(s)
		d.DispatchAsync(request(1, method, `{}`), func(r Response) {
			if r.Error == nil || r.Error.Code != "PERMISSION_DENIED" {
				t.Fatal(r)
			}
		})
		if s.calls != 0 || s.cleared != 0 {
			t.Fatal("denied request reached saver")
		}
	}
	d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
	s := &fakeTargetSaver{}
	d.SetFileSaver(s)
	token := beginSave(t, d, 0)
	for _, params := range []string{`{"token":` + jsonInt(int(token)) + `}`, `{"token":1,"target":0}`, `{"token":1,"target":null}`, `{"token":1,"target":7,"path":"C:/private"}`} {
		d.DispatchAsync(request(2, "file.commitSaveTo", params), func(r Response) {
			if r.Error == nil || r.Error.Code != "INVALID_PARAMS" {
				t.Fatal(params, r)
			}
		})
	}
	if s.calls != 0 {
		t.Fatal("invalid reuse opened target")
	}
	for _, method := range []string{"file.commitSave", "file.commitSaveAs"} {
		d.DispatchAsync(request(2, method, `{"token":1,"target":null}`), func(r Response) {
			if r.Error == nil || r.Error.Code != "INVALID_PARAMS" {
				t.Fatal("extra field accepted", r)
			}
		})
	}
}

func TestConnectedSaveRoutesAndLifecycle(t *testing.T) {
	d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
	s := &fakeTargetSaver{}
	d.SetFileSaver(s)
	token := beginSave(t, d, 0)
	raw := saveRequest(2, "file.commitSaveAs", map[string]any{"token": token})
	if !RequiresDeferred(raw) {
		t.Fatal("selection is not deferred")
	}
	d.DispatchAsync(raw, func(r Response) {
		if !r.OK {
			t.Fatal(r)
		}
	})
	s.complete(fileopen.SaveResult{Target: 7}, nil)
	token = beginSave(t, d, 0)
	raw = saveRequest(3, "file.commitSaveTo", map[string]any{"token": token, "target": 7})
	if !RequiresDeferred(raw) {
		t.Fatal("disk save is not deferred")
	}
	d.DispatchAsync(raw, func(r Response) {
		if !r.OK {
			t.Fatal(r)
		}
	})
	if s.selected != 1 || s.target != 7 {
		t.Fatal("Save reopened selection or changed target")
	}
	s.complete(fileopen.SaveResult{Target: 7}, nil)
	if r := d.Dispatch(request(4, "file.releaseSaveTarget", `{"target":8}`)); r.Error == nil || r.Error.Code != "SAVE_TARGET_INVALID" {
		t.Fatal(r)
	}
	if r := d.Dispatch(request(4, "file.releaseSaveTarget", `{"target":7}`)); !r.OK {
		t.Fatal(r)
	}
	d.DropPreparedText()
	d.Close()
	if s.cleared != 3 {
		t.Fatal("target not cleared on release, navigation and close", s.cleared)
	}
}

func TestConnectedSaveStableErrors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{{fileopen.ErrConflict, "FILE_CHANGED"}, {fileopen.ErrTarget, "SAVE_TARGET_INVALID"}, {errors.New("C:/private: failure"), "NATIVE_OPERATION_FAILED"}} {
		d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
		d.SetFileSaver(&fakeTargetSaver{fakeFileSaver: fakeFileSaver{err: tc.err}})
		token := beginSave(t, d, 0)
		d.DispatchAsync(saveRequest(2, "file.commitSaveTo", map[string]any{"token": token, "target": 7}), func(r Response) {
			if r.Error == nil || r.Error.Code != tc.code {
				t.Fatal(r)
			}
		})
		if d.savePending {
			t.Fatal("error retained pending state")
		}
	}
}
