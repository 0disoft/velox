//go:build windows

package webview2

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/0disoft/velox/internal/ipc"
	webview "github.com/jchv/go-webview2"
)

const maxWebMessageBytes = 64 << 10

type ReadyHandler func(phase string) error

type Runtime struct {
	view          webview.WebView
	dispatcher    *ipc.Dispatcher
	shutdownPhase func(name string)
	closeOnce     sync.Once
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
		StartupPhase:   config.StartupPhase,
		ShutdownPhase:  config.ShutdownPhase,
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

	runtime := &Runtime{view: view, shutdownPhase: config.ShutdownPhase}
	runtime.dispatcher = ipc.NewDispatcher(ipc.Identity{
		ID: config.AppID, Name: config.Title, Version: config.AppVersion, Platform: "windows",
	}, config.Permissions, nativeWindow{view: view, runtime: runtime})
	if err := view.SetVirtualHostNameToFolderMapping(trustedHost(config.AppID), config.AssetRoot); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("map virtual asset host: %w", err)
	}
	if err := view.Bind("__veloxInvoke", func(request json.RawMessage) ipc.Response {
		return runtime.dispatcher.Dispatch(request)
	}); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("bind native invocation bridge: %w", err)
	}
	view.Init(ipc.BridgeSource())
	if err := view.Bind("__veloxReady", onReady); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("bind ready marker: %w", err)
	}
	view.Navigate(entryURL)
	if config.StartupPhase != nil {
		config.StartupPhase("navigation-dispatched")
	}
	return runtime, nil
}

func (r *Runtime) Run() {
	r.view.Run()
	r.markShutdown("run-loop-exited")
}

func (r *Runtime) Terminate() {
	r.view.Terminate()
}

func (r *Runtime) BrowserProcessID() (uint32, error) {
	return r.view.BrowserProcessID()
}

func (r *Runtime) Close() {
	r.closeOnce.Do(func() {
		r.markShutdown("shutdown-requested")
		if r.dispatcher != nil {
			r.dispatcher.Close()
		}
		r.markShutdown("dispatcher-closed")
		// Bind callbacks enqueue their RPC response after returning. Two dispatch
		// turns let that response drain before Destroy releases WebView2 COM state.
		r.view.Dispatch(func() {
			r.view.Dispatch(r.view.Destroy)
		})
		r.markShutdown("destroy-queued")
	})
}

func (r *Runtime) markShutdown(name string) {
	if r.shutdownPhase != nil {
		r.shutdownPhase(name)
	}
}

func destroyBeforeRun(view webview.WebView) {
	view.Destroy()
	view.Run()
}
