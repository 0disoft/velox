//go:build windows

package webview2

import (
	"errors"
	"testing"
)

type bindingResponseBrowser struct {
	evaluated []string
	destroyed int
}

func (*bindingResponseBrowser) Embed(uintptr) bool { return true }
func (b *bindingResponseBrowser) Destroy()         { b.destroyed++ }
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

type destroyRunnerProbe struct {
	sequence []string
}

func (p *destroyRunnerProbe) Destroy() { p.sequence = append(p.sequence, "destroy") }
func (p *destroyRunnerProbe) Run()     { p.sequence = append(p.sequence, "run") }

type settingsProbe struct {
	contextMenuErr error
	devToolsErr    error
	calls          []string
	released       int
}

func (p *settingsProbe) PutAreDefaultContextMenusEnabled(bool) error {
	p.calls = append(p.calls, "context-menu")
	return p.contextMenuErr
}

func (p *settingsProbe) PutAreDevToolsEnabled(bool) error {
	p.calls = append(p.calls, "dev-tools")
	return p.devToolsErr
}

func (p *settingsProbe) Release() uintptr {
	p.released++
	return 0
}

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

func TestDestroyBeforeReturnPumpsNativeClose(t *testing.T) {
	probe := &destroyRunnerProbe{}
	destroyBeforeReturn(probe)

	if len(probe.sequence) != 2 || probe.sequence[0] != "destroy" || probe.sequence[1] != "run" {
		t.Fatalf("destroyBeforeReturn sequence = %v, want [destroy run]", probe.sequence)
	}
}

func TestCleanupFailedEmbedDestroysPartialBrowser(t *testing.T) {
	browser := &bindingResponseBrowser{}
	cleanupFailedEmbed(&webview{browser: browser})

	if browser.destroyed != 1 {
		t.Fatalf("partial browser destroy count = %d, want 1", browser.destroyed)
	}
}

func TestConfigureSettingsReleasesReferenceOnSuccess(t *testing.T) {
	settings := &settingsProbe{}
	if err := configureSettings(settings, false); err != nil {
		t.Fatal(err)
	}
	if settings.released != 1 {
		t.Fatalf("settings release count = %d, want 1", settings.released)
	}
	if len(settings.calls) != 2 || settings.calls[0] != "context-menu" || settings.calls[1] != "dev-tools" {
		t.Fatalf("settings calls = %v, want [context-menu dev-tools]", settings.calls)
	}
}

func TestConfigureSettingsReleasesReferenceOnFailure(t *testing.T) {
	cases := []struct {
		name           string
		contextMenuErr error
		devToolsErr    error
		wantCalls      int
	}{
		{name: "context menu", contextMenuErr: errors.New("context menu failed"), wantCalls: 1},
		{name: "developer tools", devToolsErr: errors.New("developer tools failed"), wantCalls: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings := &settingsProbe{contextMenuErr: tc.contextMenuErr, devToolsErr: tc.devToolsErr}
			if err := configureSettings(settings, false); err == nil {
				t.Fatal("configureSettings succeeded, want failure")
			}
			if settings.released != 1 {
				t.Fatalf("settings release count = %d, want 1", settings.released)
			}
			if len(settings.calls) != tc.wantCalls {
				t.Fatalf("settings calls = %v, want %d calls", settings.calls, tc.wantCalls)
			}
		})
	}
}

var _ browser = (*bindingResponseBrowser)(nil)
var _ browserSettings = (*settingsProbe)(nil)
var _ destroyRunner = (*destroyRunnerProbe)(nil)
