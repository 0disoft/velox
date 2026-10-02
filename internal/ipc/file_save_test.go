package ipc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/fileopen"
)

type fakeFileSaver struct {
	complete   func(fileopen.SaveResult, error)
	text, name string
	calls      int
	err        error
}

func (f *fakeFileSaver) Save(text, name string, done func(fileopen.SaveResult, error)) error {
	f.text, f.name, f.complete = text, name, done
	f.calls++
	return f.err
}
func saveRequest(id uint32, method string, params any) json.RawMessage {
	data, _ := json.Marshal(params)
	return request(id, method, string(data))
}
func beginSave(t *testing.T, d *Dispatcher, size int) uint32 {
	t.Helper()
	r := d.Dispatch(request(1, "file.beginSave", `{"name":"notes.md","bytes":`+jsonInt(size)+`}`))
	if !r.OK {
		t.Fatal(r)
	}
	data, _ := json.Marshal(r.Result)
	var value struct{ Token uint32 }
	json.Unmarshal(data, &value)
	return value.Token
}
func jsonInt(n int) string { data, _ := json.Marshal(n); return string(data) }

func TestFileSavePermissionAndParameters(t *testing.T) {
	for _, method := range []string{"file.beginSave", "file.appendSave", "file.commitSave", "file.cancelSave"} {
		d := NewDispatcher(Identity{}, nil, &fakeWindow{})
		s := &fakeFileSaver{}
		d.SetFileSaver(s)
		d.DispatchAsync(request(1, method, `{}`), func(r Response) {
			if r.Error == nil || r.Error.Code != "PERMISSION_DENIED" {
				t.Fatal(r)
			}
		})
		if s.calls != 0 || d.upload != nil {
			t.Fatal("denied request staged text or opened dialog")
		}
	}
	d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
	d.SetFileSaver(&fakeFileSaver{})
	for _, params := range []string{`{}`, `{"name":null,"bytes":0}`, `{"name":"a.txt","bytes":null}`, `{"name":"a.txt","bytes":-1}`, `{"name":"../a.txt","bytes":0}`, `{"name":"a.txt","bytes":0,"path":"C:/private"}`} {
		if r := d.Dispatch(request(1, "file.beginSave", params)); r.Error == nil || r.Error.Code != "INVALID_PARAMS" {
			t.Fatal(params, r)
		}
	}
	if r := d.Dispatch(request(1, "file.beginSave", `{"name":"a.txt","bytes":2097153}`)); r.Error == nil || r.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Fatal(r)
	}
}

func TestFileSaveUploadBoundsAndCompletion(t *testing.T) {
	d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
	s := &fakeFileSaver{}
	d.SetFileSaver(s)
	token := beginSave(t, d, fileopen.MaxTextBytes)
	if r := d.Dispatch(request(2, "file.beginSave", `{"name":"next.txt","bytes":0}`)); r.Error == nil || r.Error.Code != "TOO_MANY_REQUESTS" {
		t.Fatal(r)
	}
	for offset := 0; offset < fileopen.MaxTextBytes; offset += 4096 {
		raw := saveRequest(2, "file.appendSave", map[string]any{"token": token, "offset": offset, "text": strings.Repeat("x", 4096)})
		if len(raw) > MaxRequestBytes {
			t.Fatal("chunk exceeds bound")
		}
		if r := d.Dispatch(raw); !r.OK {
			t.Fatal(r)
		}
	}
	if r := d.Dispatch(saveRequest(2, "file.appendSave", map[string]any{"token": token, "offset": fileopen.MaxTextBytes, "text": "x"})); r.Error == nil {
		t.Fatal("upload overflow accepted")
	}
	replies := 0
	d.DispatchAsync(saveRequest(3, "file.commitSave", map[string]any{"token": token}), func(r Response) {
		replies++
		if !r.OK {
			t.Fatal(r)
		}
	})
	if len(s.text) != fileopen.MaxTextBytes || s.name != "notes.md" || s.calls != 1 {
		t.Fatal("wrong save body")
	}
	if r := d.Dispatch(request(3, "app.getInfo", `{}`)); r.Error == nil || r.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatal("ID released before save completion")
	}
	if r := d.Dispatch(request(4, "file.beginSave", `{"name":"next.txt","bytes":0}`)); r.Error == nil || r.Error.Code != "TOO_MANY_REQUESTS" {
		t.Fatal(r)
	}
	s.complete(fileopen.SaveResult{Cancelled: true}, nil)
	s.complete(fileopen.SaveResult{}, errors.New("duplicate"))
	oldComplete := s.complete
	if replies != 1 {
		t.Fatal("duplicate completion")
	}
	next := beginSave(t, d, 0)
	if next == token {
		t.Fatal("token reused")
	}
	// A late callback must not release a newer save's pending state.
	d.DispatchAsync(saveRequest(5, "file.commitSave", map[string]any{"token": next}), func(Response) {})
	oldComplete(fileopen.SaveResult{}, nil)
	if !d.savePending {
		t.Fatal("new save not pending")
	}
	s.complete(fileopen.SaveResult{Cancelled: true}, nil)
}

func TestFileSaveInvalidChunksAndAbandonedUploads(t *testing.T) {
	d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
	s := &fakeFileSaver{}
	d.SetFileSaver(s)
	token := beginSave(t, d, 4)
	for _, p := range []map[string]any{
		{"token": token + 1, "offset": 0, "text": "a"}, {"token": token, "offset": 1, "text": "a"},
		{"token": token, "offset": 0, "text": "a\x00b"}, {"token": token, "offset": 0, "text": "abcde"},
		{"token": token, "offset": 0, "text": "a", "path": "C:/private"},
	} {
		if r := d.Dispatch(saveRequest(2, "file.appendSave", p)); r.Error == nil {
			t.Fatal(p)
		}
	}
	d.DispatchAsync(saveRequest(3, "file.commitSave", map[string]any{"token": token}), func(r Response) {
		if r.Error == nil || r.Error.Code != "INVALID_PARAMS" {
			t.Fatal(r)
		}
	})
	if s.calls != 0 {
		t.Fatal("incomplete upload saved")
	}
	if r := d.Dispatch(saveRequest(4, "file.cancelSave", map[string]any{"token": token})); !r.OK {
		t.Fatal(r)
	}
	beginSave(t, d, 4)
	d.DropPreparedText()
	if d.upload != nil {
		t.Fatal("navigation retained text")
	}
	beginSave(t, d, 4)
	d.Close()
	if d.upload != nil {
		t.Fatal("shutdown retained text")
	}
}

func TestFileSaveErrorsAndShutdown(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{fileopen.ErrBusy, "TOO_MANY_REQUESTS"}, {fileopen.ErrTooLarge, "PAYLOAD_TOO_LARGE"}, {fileopen.ErrUnsupported, "UNSUPPORTED_FILE"}, {fileopen.ErrRecovery, "SAVE_RECOVERY_REQUIRED"}, {errors.New("C:/secret: denied"), "NATIVE_OPERATION_FAILED"},
	} {
		d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
		d.SetFileSaver(&fakeFileSaver{err: tc.err})
		token := beginSave(t, d, 0)
		d.DispatchAsync(saveRequest(2, "file.commitSave", map[string]any{"token": token}), func(r Response) {
			if r.Error == nil || r.Error.Code != tc.code || strings.Contains(r.Error.Message, "C:/secret") {
				t.Fatal(r)
			}
		})
		if d.savePending {
			t.Fatal("failed save retained busy state")
		}
	}
	d := NewDispatcher(Identity{}, []string{PermissionFileSave}, &fakeWindow{})
	s := &fakeFileSaver{}
	d.SetFileSaver(s)
	token := beginSave(t, d, 0)
	d.DispatchAsync(saveRequest(2, "file.commitSave", map[string]any{"token": token}), func(r Response) {
		if r.Error == nil || r.Error.Code != "SHUTTING_DOWN" {
			t.Fatal(r)
		}
	})
	d.Close()
	s.complete(fileopen.SaveResult{}, nil)
}
