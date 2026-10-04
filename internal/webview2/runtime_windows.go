//go:build windows

package webview2

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/0disoft/velox/internal/clipboard"
	"github.com/0disoft/velox/internal/externalurl"
	"github.com/0disoft/velox/internal/fileopen"
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

	var documentGeneration atomic.Uint64
	var runtime *Runtime
	view, createErr := webview.NewWithOptionsAndError(webview.WebViewOptions{
		Debug:                   config.Debug,
		DataPath:                config.DataPath,
		BrowserExecutableFolder: config.BrowserExecutableFolder,
		AutoFocus:               true,
		DenyAllPermissions:      true,
		FileSystemAccessAllowed: func(origin string) bool {
			return isTrustedDocument(origin, config.AppID)
		},
		MessageSourceAllowed: func(source string) bool {
			return isTrustedDocument(source, config.AppID)
		},
		MaxWebMessageBytes: maxWebMessageBytes,
		NavigationAllowed: func(uri string) bool {
			allowed := isTrustedDocument(uri, config.AppID)
			if allowed {
				documentGeneration.Add(1)
				if runtime != nil {
					runtime.dispatcher.DropPreparedText()
					runtime.dispatcher.DropFolderTarget()
				}
			}
			return allowed
		},
		DenyFrames:     true,
		DenyNewWindows: true,
		DenyDownloads:  true,
		PolicyBlocked:  config.PolicyBlocked,
		StartupPhase:   config.StartupPhase,
		ShutdownPhase:  config.ShutdownPhase,
		WindowOptions: webview.WindowOptions{
			IconId: 1,
			Title:  config.Title,
			Width:  config.Width,
			Height: config.Height,
			Center: true,
		},
	})
	if view == nil {
		if errors.Is(createErr, webview.ErrInitializationCanceled) {
			return nil, ErrInitializationCanceled
		}
		return nil, ErrRuntimeUnavailable
	}
	setSmallWindowIcon(uintptr(view.Window()))
	if err := installFilePermissionMenu(view, trustedOrigin(config.AppID)); err != nil {
		destroyBeforeRun(view)
		return nil, err
	}

	runtime = &Runtime{view: view, shutdownPhase: config.ShutdownPhase}
	runtime.dispatcher = ipc.NewDispatcher(ipc.Identity{
		ID: config.AppID, Name: config.Title, Version: config.AppVersion, Platform: "windows",
	}, config.Permissions, nativeWindow{view: view, runtime: runtime})
	if slices.Contains(config.Permissions, ipc.PermissionClipboardWrite) {
		runtime.dispatcher.SetClipboardWriter(clipboard.NewWindows(uintptr(view.Window())))
	}
	if slices.Contains(config.Permissions, ipc.PermissionClipboardRead) {
		runtime.dispatcher.SetClipboardReader(clipboard.NewWindowsReader(uintptr(view.Window()), view.Dispatch,
			func() bool { return !runtime.dispatcher.IsClosing() }, documentGeneration.Load, config.Title))
	}
	if slices.Contains(config.Permissions, ipc.PermissionFileOpen) {
		runtime.dispatcher.SetFileOpener(fileopen.NewWindows(uintptr(view.Window()), view.Dispatch,
			func() bool { return !runtime.dispatcher.IsClosing() }, documentGeneration.Load))
	}
	if slices.Contains(config.Permissions, ipc.PermissionFileSave) {
		runtime.dispatcher.SetFileSaver(fileopen.NewWindowsSaver(uintptr(view.Window()), view.Dispatch,
			func() bool { return !runtime.dispatcher.IsClosing() }, documentGeneration.Load))
	}
	if slices.Contains(config.Permissions, ipc.PermissionExternal) {
		runtime.dispatcher.SetExternalOpener(&externalurl.Scheduler{
			// Let the binding's queued response run before modal confirmation.
			Dispatch:      func(fn func()) { view.Dispatch(func() { view.Dispatch(fn) }) },
			IsClosing:     runtime.dispatcher.IsClosing,
			Opener:        externalurl.NewWindows(uintptr(view.Window()), runtime.dispatcher.IsClosing),
			NotifyFailure: func() { externalurl.NotifyWindowsFailure(uintptr(view.Window())) },
		})
	}
	if slices.Contains(config.Permissions, ipc.PermissionFolderRead) {
		runtime.dispatcher.SetFolderAccess(fileopen.NewWindowsFolder(uintptr(view.Window()), view.Dispatch,
			func() bool { return !runtime.dispatcher.IsClosing() }, documentGeneration.Load))
	}
	if err := view.SetVirtualHostNameToFolderMapping(trustedHost(config.AppID), config.AssetRoot); err != nil {
		destroyBeforeRun(view)
		return nil, fmt.Errorf("map virtual asset host: %w", err)
	}
	// The WebView2 message callback invokes this binding synchronously on the
	// UI/COM thread. Keep native window dispatch here and do not move it to a goroutine.
	if err := view.Bind("__veloxInvoke", func(request json.RawMessage) any {
		if ipc.RequiresDeferred(request) {
			generation := documentGeneration.Load()
			return &webview.DeferredResult{Start: func(complete func(any, error)) {
				if generation != documentGeneration.Load() || runtime.dispatcher.IsClosing() {
					complete(runtime.dispatcher.Dispatch(request), nil)
					return
				}
				runtime.dispatcher.DispatchAsync(request, func(response ipc.Response) {
					complete(response, nil)
				})
			}}
		}
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
	if config.RememberState {
		if err := installWindowState(uintptr(view.Window()), config.DataPath, config.AppID); err != nil {
			destroyBeforeRun(view)
			return nil, err
		}
	}
	if err := installWindowMinimums(uintptr(view.Window()), config.MinWidth, config.MinHeight); err != nil {
		destroyBeforeRun(view)
		return nil, err
	}
	if config.SingleInstance != nil {
		if err := config.SingleInstance.Attach(uintptr(view.Window())); err != nil {
			destroyBeforeRun(view)
			return nil, err
		}
	}
	if err := installSystemTray(uintptr(view.Window()), config.Title, config.Tray, runtime.dispatcher.IsClosing); err != nil {
		destroyBeforeRun(view)
		return nil, err
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
