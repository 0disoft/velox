package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionRuntimeKeepsSecurityControls(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	assertSourceMarkers(t, filepath.Join(root, "internal", "webview2", "runtime_windows.go"), []string{
		"DenyAllPermissions:      true",
		"MaxWebMessageBytes: maxWebMessageBytes",
		"DenyFrames:     true",
		"DenyNewWindows: true",
		"DenyDownloads:  true",
		"MessageSourceAllowed:",
		"ipc.BridgeSource()",
	})
	assertSourceMarkers(t, filepath.Join(root, "third_party", "go-webview2", "webview.go"), []string{
		"settings.PutAreDefaultContextMenusEnabled(options.Debug)",
		"settings.PutAreDevToolsEnabled(options.Debug)",
	})
}

func TestProductionHostOpensNoListeningSocket(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, relativeRoot := range []string{
		filepath.Join("cmd", "velox-host"),
		filepath.Join("internal", "ipc"),
		filepath.Join("internal", "webview2"),
	} {
		err := filepath.WalkDir(filepath.Join(root, relativeRoot), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, forbidden := range []string{"net.Listen(", "http.ListenAndServe(", "http.Serve("} {
				if strings.Contains(string(source), forbidden) {
					t.Errorf("%s contains forbidden listener %q", path, forbidden)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assertSourceMarkers(t *testing.T, path string, markers []string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range markers {
		if !strings.Contains(string(source), marker) {
			t.Errorf("%s is missing security marker %q", path, marker)
		}
	}
}
