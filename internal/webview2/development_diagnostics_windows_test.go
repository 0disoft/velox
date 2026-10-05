//go:build windows

package webview2

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type diagnosticView struct {
	fakeWebView
	binds   int
	scripts []string
	bound   func(json.RawMessage)
	err     error
}

func (v *diagnosticView) Bind(name string, callback interface{}) error {
	v.binds++
	if name != "__veloxDevDiagnostic" {
		return errors.New("unexpected binding")
	}
	v.bound = callback.(func(json.RawMessage))
	return v.err
}

func (v *diagnosticView) Init(script string) { v.scripts = append(v.scripts, script) }

func TestDevelopmentDiagnosticsDefaultOffAndClosingGate(t *testing.T) {
	view := &diagnosticView{}
	var output bytes.Buffer
	if err := installDevelopmentDiagnostics(view, false, "missing root", &output, func() bool { return false }); err != nil || view.binds != 0 {
		t.Fatal("disabled diagnostics installed or inspected assets")
	}
	if err := installDevelopmentDiagnostics(view, true, "missing root", nil, func() bool { return false }); err != nil || view.binds != 0 {
		t.Fatal("diagnostics without a writer installed or inspected assets")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	closing := false
	if err := installDevelopmentDiagnostics(view, true, root, &output, func() bool { return closing }); err != nil {
		t.Fatal(err)
	}
	if view.binds != 1 || len(view.scripts) != 1 {
		t.Fatal("missing debug binding/listeners")
	}
	raw := json.RawMessage(`{"kind":"uncaught-error","source":"app.js","line":3,"column":4}`)
	view.bound(raw)
	if output.String() != "velox-debug: uncaught-error app.js:3:4\n" {
		t.Fatalf("output=%q", output.String())
	}
	closing = true
	output.Reset()
	view.bound(raw)
	if output.Len() != 0 {
		t.Fatal("diagnostics ran during shutdown")
	}
}

func TestDevelopmentDiagnosticsBindingFailureDoesNotInjectListeners(t *testing.T) {
	view := &diagnosticView{err: errors.New("binding failure")}
	var output bytes.Buffer
	if err := installDevelopmentDiagnostics(view, true, t.TempDir(), &output, func() bool { return false }); err == nil || len(view.scripts) != 0 {
		t.Fatal("binding failure was ignored")
	}
}
