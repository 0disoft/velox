package main

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/webview2"
)

func TestDefaultDataPathIsStableAndAppScoped(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LocalAppData", base)
	path, err := defaultDataPath("dev.velox.hello")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "Velox", "profiles", "dev.velox.hello")
	if path != want || !filepath.IsAbs(path) {
		t.Fatalf("defaultDataPath() = %q, want %q", path, want)
	}
}

func TestPolicyAuditRequiresIPCAndEveryBrowserPolicy(t *testing.T) {
	audit := newPolicyAudit(true)
	completed := 0
	audit.complete = func() { completed++ }
	for _, policy := range []string{
		webview2.PolicyNavigation,
		webview2.PolicyFrameNavigation,
		webview2.PolicyNewWindow,
		webview2.PolicyDownload,
		webview2.PolicyPermission,
	} {
		audit.record(policy)
	}
	if completed != 0 {
		t.Fatal("policy audit completed before the trusted IPC fixture passed")
	}
	audit.markIPCReady("ipc-ok")
	audit.markIPCReady("ipc-ok")
	if completed != 1 || len(audit.missing()) != 0 {
		t.Fatalf("policy audit completed=%d missing=%v", completed, audit.missing())
	}
}
