package main

import (
	"path/filepath"
	"testing"
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
