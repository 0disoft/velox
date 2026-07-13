//go:build windows

package webview2

import (
	"testing"
	"unsafe"

	webview "github.com/jchv/go-webview2"
)

type fakeWebView struct {
	dispatches []func()
	destroyed  int
	sequence   []string
}

func (f *fakeWebView) Run() {
	f.sequence = append(f.sequence, "run")
}

func (f *fakeWebView) Terminate() {}

func (f *fakeWebView) Dispatch(callback func()) {
	f.dispatches = append(f.dispatches, callback)
}

func (f *fakeWebView) Destroy() {
	f.destroyed++
	f.sequence = append(f.sequence, "destroy")
}

func (f *fakeWebView) BrowserProcessID() (uint32, error) { return 42, nil }

func (f *fakeWebView) Window() unsafe.Pointer { return nil }

func (f *fakeWebView) SetTitle(string) {}

func (f *fakeWebView) SetSize(int, int, webview.Hint) {}

func (f *fakeWebView) Navigate(string) {}

func (f *fakeWebView) SetVirtualHostNameToFolderMapping(string, string) error { return nil }

func (f *fakeWebView) SetHtml(string) {}

func (f *fakeWebView) Init(string) {}

func (f *fakeWebView) Eval(string) {}

func (f *fakeWebView) Bind(string, interface{}) error { return nil }

func (f *fakeWebView) drainOne(t *testing.T) {
	t.Helper()
	if len(f.dispatches) == 0 {
		t.Fatal("no queued dispatch")
	}
	callback := f.dispatches[0]
	f.dispatches = f.dispatches[1:]
	callback()
}

func TestRuntimeCloseDrainsBoundResponseBeforeDestroy(t *testing.T) {
	view := &fakeWebView{}
	runtime := &Runtime{view: view}

	runtime.Close()
	runtime.Close()
	if len(view.dispatches) != 1 || view.destroyed != 0 {
		t.Fatalf("Close() queued=%d destroyed=%d, want one deferred close", len(view.dispatches), view.destroyed)
	}

	view.drainOne(t)
	if len(view.dispatches) != 1 || view.destroyed != 0 {
		t.Fatalf("first dispatch queued=%d destroyed=%d, want nested destroy", len(view.dispatches), view.destroyed)
	}

	view.drainOne(t)
	if view.destroyed != 1 || len(view.dispatches) != 0 {
		t.Fatalf("second dispatch queued=%d destroyed=%d, want one destroy", len(view.dispatches), view.destroyed)
	}
}

func TestDestroyBeforeRunProcessesNativeClose(t *testing.T) {
	view := &fakeWebView{}
	destroyBeforeRun(view)

	if len(view.sequence) != 2 || view.sequence[0] != "destroy" || view.sequence[1] != "run" {
		t.Fatalf("destroyBeforeRun sequence = %v, want [destroy run]", view.sequence)
	}
}

func TestRuntimeReportsBrowserProcessID(t *testing.T) {
	runtime := &Runtime{view: &fakeWebView{}}
	processID, err := runtime.BrowserProcessID()
	if err != nil {
		t.Fatal(err)
	}
	if processID != 42 {
		t.Fatalf("BrowserProcessID() = %d, want 42", processID)
	}
}
