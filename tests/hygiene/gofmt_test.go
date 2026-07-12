package hygiene_test

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalWebViewForkIsFormatted(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "third_party", "go-webview2"))
	if err != nil {
		t.Fatal(err)
	}
	files := []string{
		"common.go",
		"webview.go",
		filepath.Join("pkg", "edge", "chromium.go"),
		filepath.Join("pkg", "edge", "corewebview2.go"),
		filepath.Join("pkg", "edge", "ICoreWebView2Controller.go"),
		filepath.Join("pkg", "edge", "ICoreWebView2_3.go"),
		filepath.Join("pkg", "edge", "ICoreWebView2_4.go"),
		filepath.Join("pkg", "edge", "policy_events.go"),
	}
	for _, relativePath := range files {
		path := filepath.Join(root, relativePath)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		formatted, err := format.Source(source)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if !bytes.Equal(source, formatted) {
			t.Errorf("%s is not gofmt-formatted: %s", path, firstDifference(source, formatted))
		}
	}
}

func firstDifference(source, formatted []byte) string {
	limit := len(source)
	if len(formatted) < limit {
		limit = len(formatted)
	}
	for index := 0; index < limit; index++ {
		if source[index] != formatted[index] {
			end := index + 80
			if end > limit {
				end = limit
			}
			return fmt.Sprintf("first byte differs at offset %d: source=%q formatted=%q",
				index, source[index:end], formatted[index:end])
		}
	}
	return fmt.Sprintf("length differs: source=%d formatted=%d", len(source), len(formatted))
}
