//go:build windows

package webview2

import (
	"errors"
	"fmt"
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
	entryURL, err := virtualEntryURL(config.AppID, config.AssetRoot, config.EntryPath)
	if err != nil {
		return nil, err
	}

	view := webview.NewWithOptions(webview.WebViewOptions{
		Debug:              config.Debug,
		DataPath:           config.DataPath,
		AutoFocus:          true,
		DenyAllPermissions: true,
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
	if err := view.SetVirtualHostNameToFolderMapping(trustedHost(config.AppID), config.AssetRoot); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("map virtual asset host: %w", err)
	}
	if err := view.Bind("__veloxM0Ready", onReady); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("bind M0 ready marker: %w", err)
	}
	view.Navigate(entryURL)
	return runtime, nil
}

func (r *M0Runtime) Run() {
	r.view.Run()
}

func (r *M0Runtime) Terminate() {
	r.view.Terminate()
}

func (r *M0Runtime) Close() {
	r.closeOnce.Do(func() {
		// Bind callbacks enqueue their RPC response after returning. Two dispatch
		// turns let that response drain before Destroy releases WebView2 COM state.
		r.view.Dispatch(func() {
			r.view.Dispatch(r.view.Destroy)
		})
	})
}

func destroyBeforeRun(view webview.WebView) {
	view.Destroy()
	view.Run()
}
