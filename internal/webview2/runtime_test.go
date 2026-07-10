package webview2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestM0CapabilitiesDoNotClaimProductionPolicies(t *testing.T) {
	got := M0Capabilities()
	if !got.VirtualHTTPSOrigin || !got.PermissionPolicy {
		t.Fatalf("implemented M0 capabilities are missing: %+v", got)
	}
	if got.TrustedOriginMessages || got.NavigationPolicy || got.NewWindowPolicy ||
		got.DownloadPolicy || got.CleanShutdown {
		t.Fatalf("M0 capabilities claim unimplemented policies: %+v", got)
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
