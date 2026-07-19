package ipc

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeWindow struct {
	state         string
	operationErr  error
	blockMinimize <-chan struct{}
}

func (f *fakeWindow) State() (string, error) { return f.state, f.operationErr }
func (f *fakeWindow) Minimize() error {
	if f.blockMinimize != nil {
		<-f.blockMinimize
	}
	return f.operationErr
}
func (f *fakeWindow) Maximize() error { return f.operationErr }
func (f *fakeWindow) Restore() error  { return f.operationErr }
func (f *fakeWindow) Close() error    { return f.operationErr }

func TestDispatcherReturnsApplicationInfo(t *testing.T) {
	dispatcher := NewDispatcher(Identity{
		ID: "dev.velox.test", Name: "Test", Version: "1.2.3", Platform: "windows",
	}, []string{PermissionAppInfo}, &fakeWindow{})

	response := dispatcher.Dispatch(request(1, "app.getInfo", `{}`))
	if !response.OK || response.Error != nil {
		t.Fatalf("Dispatch() = %+v, want success", response)
	}
	info, ok := response.Result.(Identity)
	if !ok || info.ID != "dev.velox.test" || info.Platform != "windows" {
		t.Fatalf("result = %#v, want application identity", response.Result)
	}
}

func TestDispatcherExposesOnlyDeclaredWindowMethods(t *testing.T) {
	dispatcher := NewDispatcher(Identity{}, []string{PermissionWindow}, &fakeWindow{state: "normal"})
	for id, method := range []string{
		"window.getState", "window.minimize", "window.maximize", "window.restore", "window.close",
	} {
		response := dispatcher.Dispatch(request(uint32(id+1), method, `{}`))
		if !response.OK || response.Error != nil {
			t.Fatalf("%s Dispatch() = %+v, want success", method, response)
		}
		if method == "window.getState" && response.Result != "normal" {
			t.Fatalf("window.getState result = %#v", response.Result)
		}
	}
}

func TestDispatcherRejectsUnknownDeniedAndInvalidRequests(t *testing.T) {
	dispatcher := NewDispatcher(Identity{}, []string{PermissionAppInfo}, &fakeWindow{})
	tests := []struct {
		name string
		raw  string
		code string
	}{
		{name: "unknown method", raw: string(request(1, "shell.execute", `{}`)), code: "METHOD_NOT_FOUND"},
		{name: "permission denied", raw: string(request(2, "window.minimize", `{}`)), code: "PERMISSION_DENIED"},
		{name: "unexpected params", raw: string(request(3, "app.getInfo", `{"extra":true}`)), code: "INVALID_PARAMS"},
		{name: "unsupported version", raw: `{"v":2,"id":4,"method":"app.getInfo","params":{}}`, code: "UNSUPPORTED_VERSION"},
		{name: "zero ID", raw: `{"v":1,"id":0,"method":"app.getInfo","params":{}}`, code: "INVALID_REQUEST"},
		{name: "missing params", raw: `{"v":1,"id":5,"method":"app.getInfo"}`, code: "INVALID_PARAMS"},
		{name: "non-object params", raw: `{"v":1,"id":6,"method":"app.getInfo","params":[]}`, code: "INVALID_PARAMS"},
		{name: "unknown field", raw: `{"v":1,"id":7,"method":"app.getInfo","params":{},"extra":true}`, code: "INVALID_REQUEST"},
		{name: "duplicate field", raw: `{"v":1,"id":8,"id":9,"method":"app.getInfo","params":{}}`, code: "INVALID_REQUEST"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := dispatcher.Dispatch(json.RawMessage(test.raw))
			if response.OK || response.Error == nil || response.Error.Code != test.code {
				t.Fatalf("Dispatch() = %+v, want %s", response, test.code)
			}
		})
	}
}

func TestDispatcherBoundsPayloadAndNesting(t *testing.T) {
	dispatcher := NewDispatcher(Identity{}, []string{PermissionAppInfo}, &fakeWindow{})
	oversized := json.RawMessage(strings.Repeat(" ", MaxRequestBytes+1))
	if response := dispatcher.Dispatch(oversized); response.Error == nil || response.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("oversized Dispatch() = %+v", response)
	}

	nested := `{"v":1,"id":1,"method":"app.getInfo","params":{"value":` +
		strings.Repeat(`[`, MaxNestingDepth) + `null` + strings.Repeat(`]`, MaxNestingDepth) + `}}`
	if response := dispatcher.Dispatch(json.RawMessage(nested)); response.Error == nil || response.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("nested Dispatch() = %+v", response)
	}
}

func TestDispatcherBoundsInflightAndDuplicateIDs(t *testing.T) {
	release := make(chan struct{})
	window := &fakeWindow{blockMinimize: release}
	dispatcher := NewDispatcher(Identity{}, []string{PermissionWindow}, window)

	var wait sync.WaitGroup
	wait.Add(MaxInflight)
	for id := 1; id <= MaxInflight; id++ {
		go func(id int) {
			defer wait.Done()
			dispatcher.Dispatch(request(uint32(id), "window.minimize", `{}`))
		}(id)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		dispatcher.mu.Lock()
		count := len(dispatcher.inflight)
		dispatcher.mu.Unlock()
		if count == MaxInflight {
			break
		}
		if time.Now().After(deadline) {
			close(release)
			wait.Wait()
			t.Fatalf("inflight count = %d, want %d", count, MaxInflight)
		}
		runtime.Gosched()
	}

	if response := dispatcher.Dispatch(request(1, "window.minimize", `{}`)); response.Error == nil || response.Error.Code != "DUPLICATE_REQUEST_ID" {
		t.Fatalf("duplicate Dispatch() = %+v", response)
	}
	if response := dispatcher.Dispatch(request(MaxInflight+1, "window.minimize", `{}`)); response.Error == nil || response.Error.Code != "TOO_MANY_REQUESTS" {
		t.Fatalf("overflow Dispatch() = %+v", response)
	}

	close(release)
	wait.Wait()
}

func TestDispatcherRejectsNewRequestsDuringShutdown(t *testing.T) {
	dispatcher := NewDispatcher(Identity{}, []string{PermissionAppInfo}, &fakeWindow{})
	dispatcher.Close()
	response := dispatcher.Dispatch(request(1, "app.getInfo", `{}`))
	if response.Error == nil || response.Error.Code != "SHUTTING_DOWN" {
		t.Fatalf("Dispatch() = %+v, want SHUTTING_DOWN", response)
	}
}

func TestBridgeIsFrozenAndUsesBoundedProtocol(t *testing.T) {
	source := BridgeSource()
	for _, required := range []string{
		`Object.defineProperty(window, "velox"`,
		`Object.freeze({ invoke: Object.freeze(invoke) })`,
		`pending.size >= 64`,
		`{ v: 1, id, method, params }`,
		`configurable: false`,
		`writable: false`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("bridge is missing %q", required)
		}
	}
}

func request(id uint32, method, params string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"v":1,"id":%d,"method":%q,"params":%s}`, id, method, params))
}
