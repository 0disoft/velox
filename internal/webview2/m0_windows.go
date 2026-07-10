//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	webview "github.com/jchv/go-webview2"
)

type ReadyHandler func(phase string) error

type M0Runtime struct {
	view      webview.WebView
	closeOnce sync.Once
}

func OpenM0(config Config, onReady ReadyHandler) (*M0Runtime, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid WebView2 configuration: %w", err)
	}
	if onReady == nil {
		return nil, errors.New("ready handler is required")
	}

	view := webview.NewWithOptions(webview.WebViewOptions{
		Debug:     config.Debug,
		DataPath:  config.DataPath,
		AutoFocus: true,
		WindowOptions: webview.WindowOptions{
			Title:  config.Title,
			Width:  config.Width,
			Height: config.Height,
			Center: true,
		},
	})
	if view == nil {
		return nil, ErrRuntimeUnavailable
	}

	runtime := &M0Runtime{view: view}
	if err := view.Bind("__veloxM0Ready", onReady); err != nil {
		view.Destroy()
		return nil, fmt.Errorf("bind M0 ready marker: %w", err)
	}
	view.Navigate(fileURL(config.EntryPath))
	return runtime, nil
}

func (r *M0Runtime) Run() {
	r.view.Run()
}

func (r *M0Runtime) Terminate() {
	r.view.Terminate()
}

func (r *M0Runtime) Close() {
	r.closeOnce.Do(r.view.Destroy)
}

func fileURL(path string) string {
	slashed := filepath.ToSlash(path)
	if len(slashed) >= 2 && slashed[1] == ':' {
		slashed = "/" + slashed
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}
