//go:build windows

package webview2

import (
	"encoding/json"
	"io"

	"github.com/0disoft/velox/internal/devdiagnostic"
	webview "github.com/jchv/go-webview2"
)

func installDevelopmentDiagnostics(view webview.WebView, enabled bool, root string, writer io.Writer, isClosing func() bool) error {
	if !enabled || writer == nil {
		return nil
	}
	reporter, err := devdiagnostic.New(root, writer)
	if err != nil {
		return err
	}
	if err := view.Bind("__veloxDevDiagnostic", func(request json.RawMessage) {
		if !isClosing() {
			reporter.Report(request)
		}
	}); err != nil {
		return err
	}
	view.Init(devdiagnostic.ListenerSource())
	return nil
}
