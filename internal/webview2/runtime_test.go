package webview2

import (
	"path/filepath"
	"testing"
)

func TestM0CapabilitiesDoNotClaimProductionPolicies(t *testing.T) {
	got := M0Capabilities()
	if got.VirtualHTTPSOrigin || got.TrustedOriginMessages || got.NavigationPolicy ||
		got.NewWindowPolicy || got.DownloadPolicy || got.PermissionPolicy || got.CleanShutdown {
		t.Fatalf("M0 capabilities must remain explicitly incomplete: %+v", got)
	}
}

func TestConfigValidateRejectsRelativeRuntimePaths(t *testing.T) {
	tests := []struct {
		name      string
		dataPath  string
		entryPath string
	}{
		{name: "data path", dataPath: "profile", entryPath: filepath.Join(t.TempDir(), "index.html")},
		{name: "entry path", dataPath: t.TempDir(), entryPath: "index.html"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Config{
				Title:     "Hello",
				Width:     640,
				Height:    480,
				DataPath:  test.dataPath,
				EntryPath: test.entryPath,
			}
			if err := config.validate(); err == nil {
				t.Fatal("validate() succeeded, want an absolute-path error")
			}
		})
	}
}

func TestFileURL(t *testing.T) {
	got := fileURL(`C:\apps\hello world\index.html`)
	want := "file:///C:/apps/hello%20world/index.html"
	if got != want {
		t.Fatalf("fileURL() = %q, want %q", got, want)
	}
}
