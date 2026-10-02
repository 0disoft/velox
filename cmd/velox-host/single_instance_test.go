//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/runtimeconfig"
	"github.com/0disoft/velox/internal/singleinstance"
)

func TestDuplicateExitsBeforeWebViewOpen(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	guard, primary, err := singleinstance.Acquire("dev.velox.host-test", profile)
	if err != nil || !primary {
		t.Fatalf("primary=%t, %v", primary, err)
	}
	defer guard.Close()
	if err := os.Mkdir(filepath.Join(root, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web", "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := runtimeconfig.Config{RuntimeVersion: 1, App: runtimeconfig.App{ID: "dev.velox.host-test", Name: "Test", Version: "2", SingleInstance: true},
		Assets: runtimeconfig.Assets{Root: "web", Entry: "index.html"}, Window: runtimeconfig.Window{Width: 640, Height: 480}, Security: runtimeconfig.Security{Permissions: []string{}}}
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "velox.runtime.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VELOX_DATA_DIR", profile)
	t.Setenv("VELOX_BENCH_MODE", "1")
	t.Setenv("VELOX_BENCH_WEBVIEW2_BROWSER_DIR", filepath.Join(root, "missing-browser"))
	if code := run([]string{"--config", path}); code != 0 {
		t.Fatalf("secondary exit=%d", code)
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("secondary opened profile: %v", err)
	}
	badProfile := filepath.Join(root, "file")
	if err := os.WriteFile(badProfile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VELOX_DATA_DIR", badProfile)
	if code := run([]string{"--config", path}); code != 6 {
		t.Fatalf("mutex setup failure exit=%d", code)
	}
}
