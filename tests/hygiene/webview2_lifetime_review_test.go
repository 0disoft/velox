package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebView2LifetimeReviewKeepsOwnersFixesAndResidualRiskVisible(t *testing.T) {
	root := repositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "docs", "engineering", "09-webview2-com-lifetime-review.md"): {
			"Ownership and Release Map",
			"COM-001: Settings references were leaked",
			"COM-002: Settings-stage failures queued close without pumping it",
			"COM-003: Partial embed failure did not have one unconditional cleanup path",
			"runtime.Pinner",
			"live repeated startup and shutdown",
		},
		filepath.Join(root, "third_party", "go-webview2", "webview.go"): {
			"cleanupFailedEmbed(w)",
			"destroyBeforeReturn(w)",
			"defer settings.Release()",
			"if w.hwnd == 0",
		},
		filepath.Join(root, "third_party", "go-webview2", "pkg", "edge", "ICoreWebView2Settings.go"): {
			"func (i *ICoreWebView2Settings) AddRef() uintptr",
			"func (i *ICoreWebView2Settings) Release() uintptr",
			"uintptr(unsafe.Pointer(i))",
		},
		filepath.Join(root, "docs", "engineering", "08-m4-security-review.md"): {
			"Source-reviewed; monitoring residual risk",
			"09-webview2-com-lifetime-review.md",
		},
		filepath.Join(root, "docs", "product", "03-risk-register.md"): {
			"R-003",
			"09-webview2-com-lifetime-review.md",
			"runtime.Pinner",
		},
	}

	for path, markers := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range markers {
			if !strings.Contains(string(body), marker) {
				t.Errorf("%s lacks %q", filepath.Base(path), marker)
			}
		}
	}
}
