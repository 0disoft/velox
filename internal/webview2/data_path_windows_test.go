package webview2

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
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

func TestPrepareProfilePathUsesLegacyCompatibleLocalSpelling(t *testing.T) {
	root, err := os.MkdirTemp("", "vps-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if filepath.Dir(root) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(root), "vps-") {
			t.Error("unsafe profile fixture cleanup path")
			return
		}
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	for _, path := range []string{root, filepath.Join(root, "한글"), filepath.Join(root, strings.Repeat("a", maxProfileLinkPath-len(root)-5))} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		got, lease, err := prepareProfilePath(path, browserDataPath, func() (string, error) {
			t.Fatal("short local profile must not prepare another directory")
			return "", nil
		})
		if err != nil || lease != nil || got != filepath.Clean(path) {
			t.Fatalf("legacy browser spelling = %q, want %q; lease=%v error=%v", got, filepath.Clean(path), lease, err)
		}
		original, _ := os.Stat(browserDataPath(path))
		selected, err := os.Stat(got)
		if err != nil || !os.SameFile(original, selected) {
			t.Fatalf("profile directory changed: %v", err)
		}
	}
}

func TestPrepareProfilePathKeepsUnsupportedPaths(t *testing.T) {
	for _, path := range []string{`\\server\share\profile`, `\\.\C:\profile`, `\\server\share\` + strings.Repeat("a", 180)} {
		got, lease, err := prepareProfilePath(path, browserDataPath, func() (string, error) {
			t.Fatal("an existing short or remote path must not prepare a link root")
			return "", nil
		})
		if err != nil || lease != nil || got != browserDataPath(path) {
			t.Fatalf("existing path changed: %q %v %v", got, lease, err)
		}
	}
	got, lease, err := prepareProfilePath(`C:\`+strings.Repeat("a", 180), func(string) string { return `C:\short` }, func() (string, error) {
		t.Fatal("verified short alias must not prepare a link root")
		return "", nil
	})
	if err != nil || lease != nil || got != `C:\short` {
		t.Fatalf("existing short alias changed: %q %v", got, err)
	}
}

func TestPrepareProfilePathUsesSameDirectoryAndReleasesLease(t *testing.T) {
	fixture, profile, _ := profileLinkFixture(t)
	root := filepath.Join(fixture, "VeloxProfileLinks")
	t.Cleanup(func() { removeFixtureProfileLinks(t, root) })
	path, lease, err := prepareProfilePath(profile, browserDataPath, func() (string, error) { return fixture, nil })
	if err != nil || lease == nil {
		t.Fatalf("prepare long profile: %v", err)
	}
	defer lease.Close()
	if strings.HasPrefix(path, `\\?\`) {
		t.Fatal("short pinned link retained the IndexedDB-incompatible prefix")
	}
	original, err := os.Stat(profile)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := os.Stat(path)
	if err != nil || !os.SameFile(original, alias) {
		t.Fatalf("profile identity changed: %v", err)
	}
	if err := os.Remove(path); err == nil {
		t.Fatal("selected link was not pinned")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("selected link remained pinned: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(profile, "marker.txt")); err != nil || string(body) != "original fixture" {
		t.Fatalf("original data changed: %q %v", body, err)
	}
}

func TestPrepareProfilePathFailsWithoutRedirectOrNativeDetails(t *testing.T) {
	fixture, profile, _ := profileLinkFixture(t)
	for _, cacheDir := range []func() (string, error){
		func() (string, error) { return "", errors.New("private native path details") },
		func() (string, error) { return filepath.Join(fixture, "missing-parent"), nil },
		func() (string, error) { return "relative", nil },
	} {
		path, lease, err := prepareProfilePath(profile, browserDataPath, cacheDir)
		if path != "" || lease != nil || err != ErrProfileUnavailable {
			t.Fatalf("failure exposed a fallback or private details: %q %v %v", path, lease, err)
		}
	}
	// Unicode byte length is not Windows path length.
	unicodePath := `C:\` + strings.Repeat("\uD55C", 100)
	got, lease, err := prepareProfilePath(unicodePath, browserDataPath, func() (string, error) {
		t.Fatal("100 BMP characters still fit the UTF-16 budget")
		return "", nil
	})
	if err != nil || lease != nil || got != browserDataPath(unicodePath) {
		t.Fatalf("UTF-16 budget changed: %q %v", got, err)
	}
	for _, path := range []string{`C:\` + strings.Repeat("\U0001f600", 60), "C:\\invalid\x00profile"} {
		got, lease, err := prepareProfilePath(path, browserDataPath, func() (string, error) {
			return "", errors.New("private details")
		})
		if got != "" || lease != nil || err != ErrProfileUnavailable {
			t.Fatalf("invalid or over-budget UTF-16 path accepted: %q %v", got, err)
		}
	}
}

func removeFixtureProfileLinks(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Error(err)
		return
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		name, _ := windows.UTF16PtrFromString(browserDataPath(path))
		h, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			t.Error(err)
			return
		}
		var info windows.ByHandleFileInformation
		err = windows.GetFileInformationByHandle(h, &info)
		windows.CloseHandle(h)
		if err != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
			t.Errorf("unexpected fixture link entry: %v", err)
			return
		}
		if err := os.Remove(path); err != nil {
			t.Error(err)
			return
		}
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
