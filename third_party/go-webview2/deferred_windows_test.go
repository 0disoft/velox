//go:build windows

package webview2

import (
	"strings"
	"testing"
)

func TestDeferredBindingStartsOutsideCallbackAndCompletesOnce(t *testing.T) {
	browser := &bindingResponseBrowser{}
	var complete func(any, error)
	view := &webview{browser: browser, bindings: map[string]interface{}{
		"pick": func() *DeferredResult { return &DeferredResult{Start: func(done func(any, error)) { complete = done }} },
	}}
	view.msgcb(`{"id":1,"method":"pick","params":[],"session":"0123456789abcdef0123456789abcdef"}`)
	if complete != nil || len(view.dispatchq) != 1 {
		t.Fatal("deferred handler ran inside WebView callback")
	}
	view.dispatchq[0]()
	complete("selected", nil)
	complete("duplicate", nil)
	if len(view.dispatchq) != 2 {
		t.Fatal("response was queued more than once")
	}
	view.dispatchq[1]()
	if len(browser.evaluated) != 1 || !strings.Contains(browser.evaluated[0], `window._rpc.session === "0123456789abcdef0123456789abcdef"`) {
		t.Fatal("response is not bound to the originating document")
	}
}

func TestDeferredBindingDoesNotStartAfterClosing(t *testing.T) {
	started := false
	view := &webview{browser: &bindingResponseBrowser{}, bindings: map[string]interface{}{
		"pick": func() *DeferredResult { return &DeferredResult{Start: func(func(any, error)) { started = true }} },
	}}
	view.msgcb(`{"id":1,"method":"pick","params":[],"session":"0123456789abcdef0123456789abcdef"}`)
	view.closing = true
	view.dispatchq[0]()
	if started {
		t.Fatal("handler started after shutdown")
	}
}

func TestDeferredBindingRejectsMissingDocumentSession(t *testing.T) {
	started := false
	view := &webview{browser: &bindingResponseBrowser{}, bindings: map[string]interface{}{
		"pick": func() *DeferredResult { return &DeferredResult{Start: func(func(any, error)) { started = true }} },
	}}
	view.msgcb(`{"id":1,"method":"pick","params":[]}`)
	view.dispatchq[0]()
	if started {
		t.Fatal("handler accepted an uncorrelated document")
	}
}
