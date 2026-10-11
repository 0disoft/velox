package webview2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserDataPath(t *testing.T) {
	for _, test := range []struct{ path, want string }{
		{`C:\profile`, `\\?\C:\profile`},
		{`C:/parent/profile`, `\\?\C:\parent\profile`},
		{`C:\parent\.\discard\..\profile`, `\\?\C:\parent\profile`},
		{"C:\\profile\\\uD55C\uAE00", "\\\\?\\C:\\profile\\\uD55C\uAE00"},
		{`\\server\share\profile`, `\\?\UNC\server\share\profile`},
		{`//server/share/profile`, `\\?\UNC\server\share\profile`},
		{`\\?\C:\profile`, `\\?\C:\profile`},
		{`\\?\UNC\server\share\profile`, `\\?\UNC\server\share\profile`},
		{`\\.\C:\profile`, `\\.\C:\profile`},
		{`relative\profile`, `relative\profile`},
		{`C:profile`, `C:profile`},
		{"", ""},
	} {
		t.Run(test.path, func(t *testing.T) {
			got := browserDataPath(test.path)
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
			if again := browserDataPath(got); again != got {
				t.Fatalf("not idempotent: %q -> %q", got, again)
			}
		})
	}
}

func TestResolveBrowserDataPathPreservesDirectory(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "existing-long-profile-directory")
	if err := os.Mkdir(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved := resolveBrowserDataPath(profile)
	original, err := os.Stat(profile)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := os.Stat(resolved)
	if err != nil || !os.SameFile(original, alias) {
		t.Fatalf("profile identity changed: %q: %v", resolved, err)
	}
	marker := filepath.Join(resolved, "marker.txt")
	if err := os.WriteFile(marker, []byte("owned fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(profile, "marker.txt")); err != nil || string(data) != "owned fixture" {
		t.Fatalf("alias did not write into the original profile: %v", err)
	}
	missing := filepath.Join(root, "missing-profile")
	if got := resolveBrowserDataPath(missing); got != browserDataPath(missing) {
		t.Fatalf("missing directory substituted: %q", got)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("resolving a profile created a directory")
	}
	for _, path := range []string{`relative\profile`, `\\server\share\profile`, `\\.\C:\profile`} {
		if got := resolveBrowserDataPath(path); got != browserDataPath(path) {
			t.Fatalf("unexpected alias resolution: %q -> %q", path, got)
		}
	}
}
