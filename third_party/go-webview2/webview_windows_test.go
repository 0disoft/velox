//go:build windows

package webview2

import "testing"

type bindingResponseBrowser struct {
	evaluated []string
}

func (*bindingResponseBrowser) Embed(uintptr) bool { return true }
func (*bindingResponseBrowser) Destroy()           {}
func (*bindingResponseBrowser) BrowserProcessID() (uint32, error) {
	return 1, nil
}
func (*bindingResponseBrowser) Resize()         {}
func (*bindingResponseBrowser) Navigate(string) {}
func (*bindingResponseBrowser) SetVirtualHostNameToFolderMapping(string, string) error {
	return nil
}
func (*bindingResponseBrowser) NavigateToString(string)                  {}
func (*bindingResponseBrowser) Init(string)                              {}
func (b *bindingResponseBrowser) Eval(script string)                     { b.evaluated = append(b.evaluated, script) }
func (*bindingResponseBrowser) NotifyParentWindowPositionChanged() error { return nil }
func (*bindingResponseBrowser) Focus()                                   {}

func TestBindingResponseIsDiscardedAfterCloseBegins(t *testing.T) {
	browser := &bindingResponseBrowser{}
	view := &webview{
		browser:  browser,
		bindings: map[string]interface{}{"ready": func() {}},
	}

	view.msgcb(`{"id":1,"method":"ready","params":[]}`)
	if len(view.dispatchq) != 1 {
		t.Fatalf("queued responses = %d, want 1", len(view.dispatchq))
	}

	view.m.Lock()
	view.closing = true
	queued := view.dispatchq[0]
	view.m.Unlock()
	queued()

	if len(browser.evaluated) != 0 {
		t.Fatalf("evaluated responses = %d, want 0 after close", len(browser.evaluated))
	}
}

func TestBindingResponseIsEvaluatedWhileOpen(t *testing.T) {
	browser := &bindingResponseBrowser{}
	view := &webview{
		browser:  browser,
		bindings: map[string]interface{}{"ready": func() {}},
	}

	view.msgcb(`{"id":1,"method":"ready","params":[]}`)
	view.dispatchq[0]()

	if len(browser.evaluated) != 1 {
		t.Fatalf("evaluated responses = %d, want 1 while open", len(browser.evaluated))
	}
}

var _ browser = (*bindingResponseBrowser)(nil)
