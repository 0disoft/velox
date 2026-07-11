package webview2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeCapabilities(t *testing.T) {
	got := RuntimeCapabilities()
	if !got.VirtualHTTPSOrigin || !got.PermissionPolicy {
		t.Fatalf("implemented runtime capabilities are missing: %+v", got)
	}
	if !got.TrustedOriginMessages || !got.NavigationPolicy || !got.NewWindowPolicy || !got.DownloadPolicy {
		t.Fatalf("implemented security capabilities are missing: %+v", got)
	}
	if !got.CleanShutdown {
		t.Fatalf("implemented clean host shutdown is missing: %+v", got)
	}
}

func TestConfigValidateRejectsRelativeRuntimePaths(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "index.html")
	if err := os.WriteFile(entry, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		dataPath  string
		assetRoot string
		entryPath string
	}{
		{name: "data path", dataPath: "profile", assetRoot: root, entryPath: entry},
		{name: "asset root", dataPath: t.TempDir(), assetRoot: "web", entryPath: entry},
		{name: "entry path", dataPath: t.TempDir(), assetRoot: root, entryPath: "index.html"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Config{
				Title:     "Hello",
				AppID:     "dev.velox.hello",
				Width:     640,
				Height:    480,
				DataPath:  test.dataPath,
				AssetRoot: test.assetRoot,
				EntryPath: test.entryPath,
			}
			if err := config.validate(); err == nil {
				t.Fatal("validate() succeeded, want an absolute-path error")
			}
		})
	}
}

func TestVirtualEntryURL(t *testing.T) {
	root := filepath.Join(`C:\apps`, "hello world")
	got, err := virtualEntryURL("dev.velox.hello", root, filepath.Join(root, "nested", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	want := "https://ef242f7dd279c18b516dfdcb0078dad6.app.invalid/nested/index.html"
	if got != want {
		t.Fatalf("virtualEntryURL() = %q, want %q", got, want)
	}
}

func TestVirtualEntryURLRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := virtualEntryURL("dev.velox.hello", root, filepath.Join(root, "..", "index.html")); err == nil {
		t.Fatal("virtualEntryURL() accepted an entry outside the asset root")
	}
}

func TestIsTrustedDocument(t *testing.T) {
	appID := "dev.velox.hello"
	host := trustedHost(appID)
	tests := []struct {
		name    string
		url     string
		trusted bool
	}{
		{name: "entry", url: "https://" + host + "/index.html", trusted: true},
		{name: "query and fragment", url: "https://" + host + "/index.html?q=1#ready", trusted: true},
		{name: "remote", url: "https://example.com/", trusted: false},
		{name: "host suffix", url: "https://" + host + ".example.com/", trusted: false},
		{name: "port", url: "https://" + host + ":443/", trusted: false},
		{name: "credentials", url: "https://user@" + host + "/", trusted: false},
		{name: "http", url: "http://" + host + "/", trusted: false},
		{name: "javascript", url: "javascript:alert(1)", trusted: false},
		{name: "invalid", url: "://", trusted: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTrustedDocument(test.url, appID); got != test.trusted {
				t.Fatalf("isTrustedDocument(%q) = %t, want %t", test.url, got, test.trusted)
			}
		})
	}
}
