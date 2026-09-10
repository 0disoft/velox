//go:build windows
// +build windows

package edge

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

type Chromium struct {
	hwnd                  uintptr
	focusOnInit           bool
	controller            *ICoreWebView2Controller
	webview               *ICoreWebView2
	inited                uintptr
	envCompleted          *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler
	controllerCompleted   *iCoreWebView2CreateCoreWebView2ControllerCompletedHandler
	webMessageReceived    *iCoreWebView2WebMessageReceivedEventHandler
	permissionRequested   *iCoreWebView2PermissionRequestedEventHandler
	webResourceRequested  *iCoreWebView2WebResourceRequestedEventHandler
	acceleratorKeyPressed *ICoreWebView2AcceleratorKeyPressedEventHandler
	navigationCompleted   *ICoreWebView2NavigationCompletedEventHandler
	navigationStarting    *navigationStartingEventHandler
	frameNavigation       *navigationStartingEventHandler
	newWindowRequested    *newWindowRequestedEventHandler
	downloadStarting      *downloadStartingEventHandler

	webMessageToken               _EventRegistrationToken
	permissionToken               _EventRegistrationToken
	webResourceToken              _EventRegistrationToken
	acceleratorToken              _EventRegistrationToken
	navigationCompletedToken      _EventRegistrationToken
	navigationStartingToken       _EventRegistrationToken
	frameNavigationToken          _EventRegistrationToken
	newWindowToken                _EventRegistrationToken
	downloadToken                 _EventRegistrationToken
	webMessageRegistered          bool
	permissionRegistered          bool
	webResourceRegistered         bool
	acceleratorRegistered         bool
	navigationCompletedRegistered bool
	navigationStartingRegistered  bool
	frameNavigationRegistered     bool
	newWindowRegistered           bool
	downloadRegistered            bool
	initializationError           error
	destroyed                     bool

	environment *ICoreWebView2Environment

	// Settings
	DataPath                string
	BrowserExecutableFolder string

	// permissions
	permissions      map[CoreWebView2PermissionKind]CoreWebView2PermissionState
	globalPermission *CoreWebView2PermissionState

	// Callbacks
	MessageCallback              func(string)
	WebResourceRequestedCallback func(request *ICoreWebView2WebResourceRequest, args *ICoreWebView2WebResourceRequestedEventArgs)
	NavigationCompletedCallback  func(sender *ICoreWebView2, args *ICoreWebView2NavigationCompletedEventArgs)
	AcceleratorKeyCallback       func(uint) bool
	MessageSourceAllowed         func(source string) bool
	MaxWebMessageBytes           int
	NavigationAllowed            func(uri string) bool
	FileSystemAccessAllowed      func(origin string) bool
	DenyFrames                   bool
	DenyNewWindows               bool
	DenyDownloads                bool
	PolicyBlocked                func(kind string)
	StartupPhase                 func(name string)
	ShutdownPhase                func(name string)
}

type WebResourceResponse struct {
	Content      []byte
	StatusCode   int
	ReasonPhrase string
	Headers      string
}

type WebResourceRequestHandler func(uri string) (WebResourceResponse, bool)

func NewChromium() *Chromium {
	e := &Chromium{}
	// Embed pins the native-visible callback graph before publishing any pointer.
	e.envCompleted = newICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler(e)
	e.controllerCompleted = newICoreWebView2CreateCoreWebView2ControllerCompletedHandler(e)
	e.webMessageReceived = newICoreWebView2WebMessageReceivedEventHandler(e)
	e.permissionRequested = newICoreWebView2PermissionRequestedEventHandler(e)
	e.webResourceRequested = newICoreWebView2WebResourceRequestedEventHandler(e)
	e.acceleratorKeyPressed = newICoreWebView2AcceleratorKeyPressedEventHandler(e)
	e.navigationCompleted = newICoreWebView2NavigationCompletedEventHandler(e)
	e.navigationStarting = newNavigationStartingEventHandler(e, false)
	e.frameNavigation = newNavigationStartingEventHandler(e, true)
	e.newWindowRequested = newNewWindowRequestedEventHandler(e)
	e.downloadStarting = newDownloadStartingEventHandler(e)
	e.permissions = make(map[CoreWebView2PermissionKind]CoreWebView2PermissionState)

	return e
}

func (e *Chromium) Embed(hwnd uintptr) bool {
	if e.destroyed {
		return false
	}
	e.hwnd = hwnd

	dataPath := e.DataPath
	if dataPath == "" {
		currentExePath := make([]uint16, windows.MAX_PATH)
		_, err := windows.GetModuleFileName(windows.Handle(0), &currentExePath[0], windows.MAX_PATH)
		if err != nil {
			// What to do here?
			return false
		}
		currentExeName := filepath.Base(windows.UTF16ToString(currentExePath))
		dataPath = filepath.Join(os.Getenv("AppData"), currentExeName)
	}

	var browserExecutableFolder *uint16
	if e.BrowserExecutableFolder != "" {
		browserExecutableFolder = windows.StringToUTF16Ptr(e.BrowserExecutableFolder)
	}
	if e.StartupPhase != nil {
		e.StartupPhase("environment-create-started")
	}
	if !e.retainCallbackOwner() {
		return false
	}
	res, err := createCoreWebView2EnvironmentWithOptions(browserExecutableFolder, windows.StringToUTF16Ptr(dataPath), 0, e.envCompleted)
	if err != nil {
		log.Printf("Error calling Webview2Loader: %v", err)
		e.Destroy()
		return false
	} else if res != 0 {
		log.Printf("Result: %08x", res)
		e.Destroy()
		return false
	}
	var msg w32.Msg
	for {
		if atomic.LoadUintptr(&e.inited) != 0 {
			break
		}
		r, _, _ := w32.User32GetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		if r == 0 {
			break
		}
		_, _, _ = w32.User32TranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		_, _, _ = w32.User32DispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return e.finishInitialization()
}

func (e *Chromium) finishInitialization() bool {
	if e.destroyed || e.initializationError != nil || e.webview == nil || atomic.LoadUintptr(&e.inited) == 0 {
		e.Destroy()
		return false
	}
	e.Init("window.external={invoke:s=>window.chrome.webview.postMessage(s)}")
	return true
}

func (e *Chromium) Navigate(url string) {
	_, _, _ = e.webview.vtbl.Navigate.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(url))),
	)
}

func (e *Chromium) SetVirtualHostNameToFolderMapping(hostName, folderPath string) error {
	webview3 := e.GetICoreWebView2_3()
	if webview3 == nil {
		return windows.ERROR_NOT_SUPPORTED
	}
	defer webview3.Release()
	return webview3.SetVirtualHostNameToFolderMapping(
		hostName,
		folderPath,
		COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_DENY_CORS,
	)
}

func (e *Chromium) Destroy() {
	if e.destroyed {
		return
	}
	e.destroyed = true
	defer e.releaseCallbackOwner()
	e.markShutdown("chromium-destroy-entered")
	e.removeEventHandlers()
	e.markShutdown("event-handlers-removed")
	if e.controller != nil {
		_ = e.controller.Close()
		e.markShutdown("controller-closed")
	}
	if e.webview != nil {
		e.webview.Release()
		e.webview = nil
		e.markShutdown("webview-released")
	}
	if e.controller != nil {
		e.controller.Release()
		e.controller = nil
		e.markShutdown("controller-released")
	}
	if e.environment != nil {
		e.environment.Release()
		e.environment = nil
		e.markShutdown("environment-released")
	}
}

func (e *Chromium) markShutdown(name string) {
	if e.ShutdownPhase != nil {
		e.ShutdownPhase(name)
	}
}

func (e *Chromium) BrowserProcessID() (uint32, error) {
	if e.webview == nil {
		return 0, errors.New("WebView2 is not initialized")
	}
	return e.webview.GetBrowserProcessID()
}

func (e *Chromium) NavigateToString(htmlContent string) {
	_, _, _ = e.webview.vtbl.NavigateToString.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(htmlContent))),
	)
}

func (e *Chromium) Init(script string) {
	_, _, _ = e.webview.vtbl.AddScriptToExecuteOnDocumentCreated.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(script))),
		0,
	)
}

func (e *Chromium) Eval(script string) {
	_script, err := windows.UTF16PtrFromString(script)
	if err != nil {
		log.Fatal(err)
	}

	_, _, _ = e.webview.vtbl.ExecuteScript.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(_script)),
		0,
	)
}

func (e *Chromium) Show() error {
	return e.controller.PutIsVisible(true)
}

func (e *Chromium) Hide() error {
	return e.controller.PutIsVisible(false)
}

func (e *Chromium) QueryInterface(_, _ uintptr) uintptr {
	// Chromium is the Go owner, not a native COM interface. Each callback
	// supplies its own identity and performs reference acquisition.
	return 0x80004002
}

func (e *Chromium) EnvironmentCompleted(res uintptr, env *ICoreWebView2Environment) uintptr {
	if e.destroyed {
		return 0
	}
	if hresult(res) != nil || env == nil {
		e.initializationError = fmt.Errorf("creating environment failed with %08x", res)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	_, _, _ = env.vtbl.AddRef.Call(uintptr(unsafe.Pointer(env)))
	e.environment = env
	if e.StartupPhase != nil {
		e.StartupPhase("environment-created")
	}
	if e.destroyed {
		return 0
	}

	result, _, _ := env.vtbl.CreateCoreWebView2Controller.Call(
		uintptr(unsafe.Pointer(env)),
		e.hwnd,
		uintptr(unsafe.Pointer(e.controllerCompleted)),
	)
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("create controller: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
	}
	return 0
}

func (e *Chromium) CreateCoreWebView2ControllerCompleted(res uintptr, controller *ICoreWebView2Controller) uintptr {
	if e.destroyed {
		if hresult(res) == nil && controller != nil {
			_ = controller.Close()
		}
		return 0
	}
	if hresult(res) != nil || controller == nil {
		e.initializationError = fmt.Errorf("creating controller failed with %08x", res)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	_, _, _ = controller.vtbl.AddRef.Call(uintptr(unsafe.Pointer(controller)))
	e.controller = controller

	result, _, _ := controller.vtbl.GetCoreWebView2.Call(
		uintptr(unsafe.Pointer(controller)),
		uintptr(unsafe.Pointer(&e.webview)),
	)
	if err := hresult(result); err != nil || e.webview == nil {
		if err == nil {
			err = errors.New("controller returned an empty WebView")
		}
		e.initializationError = fmt.Errorf("get WebView: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	result, _, _ = e.webview.vtbl.AddWebMessageReceived.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(e.webMessageReceived)),
		uintptr(unsafe.Pointer(&e.webMessageToken)),
	)
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("register WebMessage policy: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	e.webMessageRegistered = true
	result, _, _ = e.webview.vtbl.AddPermissionRequested.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(e.permissionRequested)),
		uintptr(unsafe.Pointer(&e.permissionToken)),
	)
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("register permission policy: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	e.permissionRegistered = true
	result, _, _ = e.webview.vtbl.AddWebResourceRequested.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(e.webResourceRequested)),
		uintptr(unsafe.Pointer(&e.webResourceToken)),
	)
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("register WebResource handler: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	e.webResourceRegistered = true
	result, _, _ = e.webview.vtbl.AddNavigationCompleted.Call(
		uintptr(unsafe.Pointer(e.webview)),
		uintptr(unsafe.Pointer(e.navigationCompleted)),
		uintptr(unsafe.Pointer(&e.navigationCompletedToken)),
	)
	if err := hresult(result); err != nil {
		e.initializationError = fmt.Errorf("register navigation completion: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	e.navigationCompletedRegistered = true
	if err := e.controller.AddAcceleratorKeyPressed(e.acceleratorKeyPressed, &e.acceleratorToken); err != nil {
		e.initializationError = fmt.Errorf("register accelerator handler: %w", err)
		atomic.StoreUintptr(&e.inited, 1)
		return 0
	}
	e.acceleratorRegistered = true
	e.registerSecurityPolicyHandlers()
	if e.StartupPhase != nil {
		e.StartupPhase("controller-created")
	}
	if e.destroyed {
		return 0
	}

	atomic.StoreUintptr(&e.inited, 1)

	if e.focusOnInit {
		e.Focus()
	}

	return 0
}

func (e *Chromium) MessageReceived(sender *ICoreWebView2, args *iCoreWebView2WebMessageReceivedEventArgs) uintptr {
	source, err := args.Source()
	if err != nil {
		log.Printf("read WebMessage source: %v", err)
		return 0
	}
	if e.MessageSourceAllowed != nil && !e.MessageSourceAllowed(source) {
		e.reportPolicyBlocked("message-source")
		return 0
	}
	var message *uint16
	result, _, _ := args.vtbl.TryGetWebMessageAsString.Call(
		uintptr(unsafe.Pointer(args)),
		uintptr(unsafe.Pointer(&message)),
	)
	if err := hresult(result); err != nil {
		e.setPolicyError(fmt.Errorf("read WebMessage: %w", err))
		return 0
	}
	if message == nil {
		e.setPolicyError(errors.New("read WebMessage: empty native message"))
		return 0
	}
	decoded := w32.Utf16PtrToString(message)
	if e.MaxWebMessageBytes > 0 && len(decoded) > e.MaxWebMessageBytes {
		e.reportPolicyBlocked("message-size")
		windows.CoTaskMemFree(unsafe.Pointer(message))
		return 0
	}
	if e.MessageCallback != nil {
		e.MessageCallback(decoded)
	}
	_, _, _ = sender.vtbl.PostWebMessageAsString.Call(
		uintptr(unsafe.Pointer(sender)),
		uintptr(unsafe.Pointer(message)),
	)
	windows.CoTaskMemFree(unsafe.Pointer(message))
	return 0
}

func (e *Chromium) handleNavigationStarting(args *iCoreWebView2NavigationStartingEventArgs, frame bool) {
	deny := frame && e.DenyFrames
	if !frame && e.NavigationAllowed != nil {
		uri, err := args.URI()
		if err != nil {
			e.setPolicyError(fmt.Errorf("read navigation URI: %w", err))
			deny = true
		} else {
			deny = !e.NavigationAllowed(uri)
		}
	}
	if deny {
		if err := args.PutCancel(true); err != nil {
			e.setPolicyError(fmt.Errorf("cancel navigation: %w", err))
		} else if frame {
			e.reportPolicyBlocked("frame-navigation")
		} else {
			e.reportPolicyBlocked("navigation")
		}
	}
}

func (e *Chromium) SetPermission(kind CoreWebView2PermissionKind, state CoreWebView2PermissionState) {
	e.permissions[kind] = state
}

func (e *Chromium) SetGlobalPermission(state CoreWebView2PermissionState) {
	e.globalPermission = &state
}

func (e *Chromium) PermissionRequested(_ *ICoreWebView2, args *iCoreWebView2PermissionRequestedEventArgs) uintptr {
	var kind CoreWebView2PermissionKind
	resultCode, _, _ := args.vtbl.GetPermissionKind.Call(
		uintptr(unsafe.Pointer(args)),
		uintptr(unsafe.Pointer(&kind)),
	)
	result := CoreWebView2PermissionStateDeny
	if err := hresult(resultCode); err != nil {
		e.setPolicyError(fmt.Errorf("read permission kind: %w", err))
	} else if kind == CoreWebView2PermissionKindFileReadWrite && e.FileSystemAccessAllowed != nil {
		if e.fileSystemAccessUsesBrowserConsent(args) {
			result = CoreWebView2PermissionStateDefault
		}
	} else if e.globalPermission != nil {
		result = *e.globalPermission
	} else {
		var ok bool
		result, ok = e.permissions[kind]
		if !ok {
			result = CoreWebView2PermissionStateDefault
		}
	}
	resultCode, _, _ = args.vtbl.PutState.Call(
		uintptr(unsafe.Pointer(args)),
		uintptr(result),
	)
	if err := hresult(resultCode); err != nil {
		e.setPolicyError(fmt.Errorf("deny permission: %w", err))
		return 0
	}
	if result == CoreWebView2PermissionStateDeny {
		e.reportPolicyBlocked("permission")
	}
	return 0
}

func (e *Chromium) WebResourceRequested(sender *ICoreWebView2, args *ICoreWebView2WebResourceRequestedEventArgs) uintptr {
	req, err := args.GetRequest()
	if err != nil {
		log.Printf("get WebResourceRequested request: %v", err)
		return 0
	}
	defer req.Release()
	if e.WebResourceRequestedCallback != nil {
		e.WebResourceRequestedCallback(req, args)
	}
	return 0
}

func (e *Chromium) SetWebResourceRequestHandler(filter string, handler WebResourceRequestHandler) error {
	if e.webview == nil || e.environment == nil {
		return errors.New("WebView2 is not initialized")
	}
	if filter == "" || handler == nil {
		return errors.New("web resource filter and handler are required")
	}
	e.WebResourceRequestedCallback = func(request *ICoreWebView2WebResourceRequest, args *ICoreWebView2WebResourceRequestedEventArgs) {
		uri, err := request.GetUri()
		if err != nil {
			log.Printf("read WebResourceRequested URI: %v", err)
			return
		}
		resource, handled := handler(uri)
		if !handled {
			return
		}
		response, err := e.environment.CreateWebResourceResponse(
			resource.Content, resource.StatusCode, resource.ReasonPhrase, resource.Headers,
		)
		if err != nil {
			log.Printf("create WebResourceRequested response: %v", err)
			return
		}
		defer response.Release()
		if err := args.PutResponse(response); err != nil {
			log.Printf("set WebResourceRequested response: %v", err)
		}
	}
	return e.webview.AddWebResourceRequestedFilter(filter, COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL)
}

func (e *Chromium) AddWebResourceRequestedFilter(filter string, ctx COREWEBVIEW2_WEB_RESOURCE_CONTEXT) {
	err := e.webview.AddWebResourceRequestedFilter(filter, ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Chromium) Environment() *ICoreWebView2Environment {
	return e.environment
}

func (e *Chromium) registerSecurityPolicyHandlers() {
	if e.NavigationAllowed != nil {
		result, _, _ := e.webview.vtbl.AddNavigationStarting.Call(
			uintptr(unsafe.Pointer(e.webview)),
			uintptr(unsafe.Pointer(e.navigationStarting)),
			uintptr(unsafe.Pointer(&e.navigationStartingToken)),
		)
		if err := hresult(result); err != nil {
			e.initializationError = fmt.Errorf("register navigation policy: %w", err)
			return
		}
		e.navigationStartingRegistered = true
	}
	if e.DenyFrames {
		result, _, _ := e.webview.vtbl.AddFrameNavigationStarting.Call(
			uintptr(unsafe.Pointer(e.webview)),
			uintptr(unsafe.Pointer(e.frameNavigation)),
			uintptr(unsafe.Pointer(&e.frameNavigationToken)),
		)
		if err := hresult(result); err != nil {
			e.initializationError = fmt.Errorf("register frame policy: %w", err)
			return
		}
		e.frameNavigationRegistered = true
	}
	if e.DenyNewWindows {
		result, _, _ := e.webview.vtbl.AddNewWindowRequested.Call(
			uintptr(unsafe.Pointer(e.webview)),
			uintptr(unsafe.Pointer(e.newWindowRequested)),
			uintptr(unsafe.Pointer(&e.newWindowToken)),
		)
		if err := hresult(result); err != nil {
			e.initializationError = fmt.Errorf("register popup policy: %w", err)
			return
		}
		e.newWindowRegistered = true
	}
	if e.DenyDownloads {
		webview4 := e.webview.GetICoreWebView2_4()
		if webview4 == nil {
			e.initializationError = errors.New("WebView2 download policy interface is unavailable")
			return
		}
		defer webview4.Release()
		if err := webview4.AddDownloadStarting(e.downloadStarting, &e.downloadToken); err != nil {
			e.initializationError = fmt.Errorf("register download policy: %w", err)
			return
		}
		e.downloadRegistered = true
	}
}

func (e *Chromium) removeEventHandlers() {
	if e.webview == nil {
		return
	}
	if e.navigationStartingRegistered {
		_, _, _ = e.webview.vtbl.RemoveNavigationStarting.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.navigationStartingToken.Value))
		e.navigationStartingRegistered = false
	}
	if e.frameNavigationRegistered {
		_, _, _ = e.webview.vtbl.RemoveFrameNavigationStarting.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.frameNavigationToken.Value))
		e.frameNavigationRegistered = false
	}
	if e.newWindowRegistered {
		_, _, _ = e.webview.vtbl.RemoveNewWindowRequested.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.newWindowToken.Value))
		e.newWindowRegistered = false
	}
	if e.downloadRegistered {
		if webview4 := e.webview.GetICoreWebView2_4(); webview4 != nil {
			_ = webview4.RemoveDownloadStarting(e.downloadToken)
			webview4.Release()
		}
		e.downloadRegistered = false
	}
	if e.webMessageRegistered {
		_, _, _ = e.webview.vtbl.RemoveWebMessageReceived.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.webMessageToken.Value))
		e.webMessageRegistered = false
	}
	if e.permissionRegistered {
		_, _, _ = e.webview.vtbl.RemovePermissionRequested.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.permissionToken.Value))
		e.permissionRegistered = false
	}
	if e.webResourceRegistered {
		_, _, _ = e.webview.vtbl.RemoveWebResourceRequested.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.webResourceToken.Value))
		e.webResourceRegistered = false
	}
	if e.navigationCompletedRegistered {
		_, _, _ = e.webview.vtbl.RemoveNavigationCompleted.Call(
			uintptr(unsafe.Pointer(e.webview)), uintptr(e.navigationCompletedToken.Value))
		e.navigationCompletedRegistered = false
	}
	if e.controller != nil && e.acceleratorRegistered {
		_, _, _ = e.controller.vtbl.RemoveAcceleratorKeyPressed.Call(
			uintptr(unsafe.Pointer(e.controller)), uintptr(e.acceleratorToken.Value))
		e.acceleratorRegistered = false
	}
}

func (e *Chromium) setPolicyError(err error) {
	log.Printf("WebView2 policy error: %v", err)
}

func (e *Chromium) reportPolicyBlocked(kind string) {
	if e.PolicyBlocked != nil {
		e.PolicyBlocked(kind)
	}
}

// AcceleratorKeyPressed is called when an accelerator key is pressed.
// If the AcceleratorKeyCallback method has been set, it will defer handling of the keypress
// to the callback. That callback returns a bool indicating if the event was handled.
func (e *Chromium) AcceleratorKeyPressed(sender *ICoreWebView2Controller, args *ICoreWebView2AcceleratorKeyPressedEventArgs) uintptr {
	if e.AcceleratorKeyCallback == nil {
		return 0
	}
	eventKind, _ := args.GetKeyEventKind()
	if eventKind == COREWEBVIEW2_KEY_EVENT_KIND_KEY_DOWN ||
		eventKind == COREWEBVIEW2_KEY_EVENT_KIND_SYSTEM_KEY_DOWN {
		virtualKey, _ := args.GetVirtualKey()
		status, _ := args.GetPhysicalKeyStatus()
		if !status.WasKeyDown {
			_ = args.PutHandled(e.AcceleratorKeyCallback(virtualKey))
			return 0
		}
	}
	_ = args.PutHandled(false)
	return 0
}

func (e *Chromium) GetSettings() (*ICoreWebViewSettings, error) {
	return e.webview.GetSettings()
}

func (e *Chromium) GetController() *ICoreWebView2Controller {
	return e.controller
}

func boolToInt(input bool) int {
	if input {
		return 1
	}
	return 0
}

func (e *Chromium) NavigationCompleted(sender *ICoreWebView2, args *ICoreWebView2NavigationCompletedEventArgs) uintptr {
	if e.NavigationCompletedCallback != nil {
		e.NavigationCompletedCallback(sender, args)
	}
	return 0
}

func (e *Chromium) NotifyParentWindowPositionChanged() error {
	//It looks like the wndproc function is called before the controller initialization is complete.
	//Because of this the controller is nil
	if e.controller == nil {
		return nil
	}
	return e.controller.NotifyParentWindowPositionChanged()
}

func (e *Chromium) Focus() {
	if e.controller == nil {
		e.focusOnInit = true
		return
	}
	_ = e.controller.MoveFocus(COREWEBVIEW2_MOVE_FOCUS_REASON_PROGRAMMATIC)
}
