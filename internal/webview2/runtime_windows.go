//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"sync"

	webview "github.com/jchv/go-webview2"
)

const maxWebMessageBytes = 64 << 10

type ReadyHandler func(phase string) error

type Runtime struct {
	view      webview.WebView
	closeOnce sync.Once
}

func Open(config Config, onReady ReadyHandler) (*Runtime, error) {
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
		Debug:                   config.Debug,
		DataPath:                config.DataPath,
		BrowserExecutableFolder: config.BrowserExecutableFolder,
		AutoFocus:               true,
		DenyAllPermissions:      true,
		MessageSourceAllowed: func(source string) bool {
			return isTrustedDocument(source, config.AppID)
		},
		MaxWebMessageBytes: maxWebMessageBytes,
		NavigationAllowed: func(uri string) bool {
			return isTrustedDocument(uri, config.AppID)
		},
		DenyFrames:     true,
		DenyNewWindows: true,
		DenyDownloads:  true,
		PolicyBlocked:  config.PolicyBlocked,
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

	runtime := &Runtime{view: view}
	if err := view.SetVirtualHostNameToFolderMapping(trustedHost(config.AppID), config.AssetRoot); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("map virtual asset host: %w", err)
	}
	if err := view.Bind("__veloxReady", onReady); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("bind ready marker: %w", err)
	}
	view.Navigate(entryURL)
	return runtime, nil
}

func (r *Runtime) Run() {
	r.view.Run()
}

func (r *Runtime) Terminate() {
	r.view.Terminate()
}

func (r *Runtime) BrowserProcessID() (uint32, error) {
	return r.view.BrowserProcessID()
}

func (r *Runtime) Close() {
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
