#include <windows.h>
#include <objbase.h>
#include <shellapi.h>

#ifndef interface
#define interface struct
#endif

#include <WebView2.h>

extern "C" void* memset(void* destination, int value, size_t count) {
  auto* bytes = static_cast<unsigned char*>(destination);
  for (size_t index = 0; index < count; ++index) {
    bytes[index] = static_cast<unsigned char>(value);
  }
  return destination;
}

extern "C" void* memcpy(void* destination, const void* source, size_t count) {
  auto* target = static_cast<unsigned char*>(destination);
  const auto* input = static_cast<const unsigned char*>(source);
  for (size_t index = 0; index < count; ++index) {
    target[index] = input[index];
  }
  return destination;
}

namespace {

constexpr wchar_t kWindowClass[] = L"VeloxReferenceCppWindow";
constexpr wchar_t kTrustedOrigin[] = L"https://appassets.example/";
constexpr wchar_t kEntryUrl[] = L"https://appassets.example/index.html";
constexpr char kReadyMessage[] = "ready dom-2raf\n";

HWND g_window = nullptr;
ICoreWebView2Controller* g_controller = nullptr;
ICoreWebView2* g_webview = nullptr;
wchar_t g_asset_root[32768] = {};
wchar_t g_data_path[32768] = {};
wchar_t g_pipe_path[32768] = {};

bool StartsWith(const wchar_t* value, const wchar_t* prefix) noexcept {
  if (value == nullptr || prefix == nullptr) {
    return false;
  }
  while (*prefix != L'\0') {
    if (*value != *prefix) {
      return false;
    }
    ++value;
    ++prefix;
  }
  return true;
}

bool Equals(const wchar_t* left, const wchar_t* right) noexcept {
  if (left == nullptr || right == nullptr) {
    return false;
  }
  while (*left != L'\0' && *right != L'\0') {
    if (*left != *right) {
      return false;
    }
    ++left;
    ++right;
  }
  return *left == *right;
}

HRESULT QuerySingle(IUnknown* self, REFIID requested, REFIID own, void** result) noexcept {
  if (result == nullptr) {
    return E_POINTER;
  }
  *result = nullptr;
  if (IsEqualIID(requested, IID_IUnknown) || IsEqualIID(requested, own)) {
    *result = self;
    self->AddRef();
    return S_OK;
  }
  return E_NOINTERFACE;
}

ULONG StableAddRef() noexcept { return 2; }
ULONG StableRelease() noexcept { return 1; }

void WriteDiagnostic(const wchar_t* message) noexcept {
  HANDLE stderr_handle = GetStdHandle(STD_ERROR_HANDLE);
  if (stderr_handle != nullptr && stderr_handle != INVALID_HANDLE_VALUE) {
    char utf8[512] = {};
    const int length = WideCharToMultiByte(CP_UTF8, 0, message, -1, utf8,
                                           static_cast<int>(sizeof(utf8)), nullptr, nullptr);
    if (length <= 1) {
      return;
    }
    DWORD written = 0;
    WriteFile(stderr_handle, utf8, static_cast<DWORD>(length - 1), &written, nullptr);
    constexpr char newline[] = "\r\n";
    WriteFile(stderr_handle, newline, static_cast<DWORD>(sizeof(newline) - 1), &written, nullptr);
  }
}

void FailAndQuit(const wchar_t* message) noexcept {
  WriteDiagnostic(message);
  PostQuitMessage(1);
}

void NotifyReady() noexcept {
  const DWORD length = GetEnvironmentVariableW(L"VELOX_BENCH_PIPE", g_pipe_path, 32768);
  if (length > 0 && length < 32768) {
    HANDLE pipe = CreateFileW(g_pipe_path, GENERIC_WRITE, 0, nullptr, OPEN_EXISTING, 0, nullptr);
    if (pipe != INVALID_HANDLE_VALUE) {
      DWORD written = 0;
      WriteFile(pipe, kReadyMessage, static_cast<DWORD>(sizeof(kReadyMessage) - 1), &written, nullptr);
      CloseHandle(pipe);
    }
  }

  wchar_t exit_after_ready[2] = {};
  if (GetEnvironmentVariableW(L"VELOX_BENCH_EXIT_AFTER_READY", exit_after_ready, 2) == 1 &&
      exit_after_ready[0] == L'1') {
    PostQuitMessage(0);
  }
}

struct WebMessageHandler final : ICoreWebView2WebMessageReceivedEventHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2WebMessageReceivedEventHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2*, ICoreWebView2WebMessageReceivedEventArgs* args) override {
    if (args == nullptr) {
      return E_POINTER;
    }
    LPWSTR message = nullptr;
    const HRESULT result = args->TryGetWebMessageAsString(&message);
    if (SUCCEEDED(result) && Equals(message, L"ready dom-2raf")) {
      NotifyReady();
    }
    CoTaskMemFree(message);
    return result;
  }
};

struct NavigationHandler final : ICoreWebView2NavigationStartingEventHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2NavigationStartingEventHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2*, ICoreWebView2NavigationStartingEventArgs* args) override {
    if (args == nullptr) {
      return E_POINTER;
    }
    LPWSTR uri = nullptr;
    const HRESULT result = args->get_Uri(&uri);
    if (SUCCEEDED(result) && !StartsWith(uri, kTrustedOrigin)) {
      args->put_Cancel(TRUE);
    }
    CoTaskMemFree(uri);
    return result;
  }
};

struct NavigationCompletedHandler final : ICoreWebView2NavigationCompletedEventHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2NavigationCompletedEventHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2*, ICoreWebView2NavigationCompletedEventArgs* args) override {
    if (args == nullptr) {
      return E_POINTER;
    }
    BOOL success = FALSE;
    const HRESULT result = args->get_IsSuccess(&success);
    if (FAILED(result) || !success) {
      WriteDiagnostic(L"Navigation failed.");
    }
    return result;
  }
};

struct NewWindowHandler final : ICoreWebView2NewWindowRequestedEventHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2NewWindowRequestedEventHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2*, ICoreWebView2NewWindowRequestedEventArgs* args) override {
    return args == nullptr ? E_POINTER : args->put_Handled(TRUE);
  }
};

struct PermissionHandler final : ICoreWebView2PermissionRequestedEventHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2PermissionRequestedEventHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2*, ICoreWebView2PermissionRequestedEventArgs* args) override {
    return args == nullptr ? E_POINTER : args->put_State(COREWEBVIEW2_PERMISSION_STATE_DENY);
  }
};

WebMessageHandler g_web_message_handler;
NavigationHandler g_navigation_handler;
NavigationCompletedHandler g_navigation_completed_handler;
NewWindowHandler g_new_window_handler;
PermissionHandler g_permission_handler;

void ResizeController() noexcept {
  if (g_controller == nullptr || g_window == nullptr) {
    return;
  }
  RECT bounds = {};
  GetClientRect(g_window, &bounds);
  g_controller->put_Bounds(bounds);
}

struct ControllerHandler final : ICoreWebView2CreateCoreWebView2ControllerCompletedHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2CreateCoreWebView2ControllerCompletedHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(HRESULT error_code, ICoreWebView2Controller* controller) override {
    if (FAILED(error_code) || controller == nullptr) {
      FailAndQuit(L"WebView2 controller creation failed.");
      return FAILED(error_code) ? error_code : E_FAIL;
    }

    g_controller = controller;
    g_controller->AddRef();
    HRESULT result = g_controller->get_CoreWebView2(&g_webview);
    if (FAILED(result) || g_webview == nullptr) {
      FailAndQuit(L"WebView2 core acquisition failed.");
      return FAILED(result) ? result : E_FAIL;
    }
    ResizeController();

    ICoreWebView2Settings* settings = nullptr;
    if (SUCCEEDED(g_webview->get_Settings(&settings)) && settings != nullptr) {
      settings->put_AreDevToolsEnabled(FALSE);
      settings->put_AreDefaultContextMenusEnabled(FALSE);
      settings->put_IsStatusBarEnabled(FALSE);
      settings->Release();
    }

    ICoreWebView2_3* webview3 = nullptr;
    result = g_webview->QueryInterface(IID_PPV_ARGS(&webview3));
    if (FAILED(result) || webview3 == nullptr) {
      FailAndQuit(L"WebView2 virtual host mapping is unavailable.");
      return FAILED(result) ? result : E_NOINTERFACE;
    }
    result = webview3->SetVirtualHostNameToFolderMapping(
        L"appassets.example", g_asset_root, COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_DENY_CORS);
    webview3->Release();
    if (FAILED(result)) {
      FailAndQuit(L"WebView2 virtual host mapping failed.");
      return result;
    }

    EventRegistrationToken token = {};
    g_webview->add_WebMessageReceived(&g_web_message_handler, &token);
    g_webview->add_NavigationStarting(&g_navigation_handler, &token);
    g_webview->add_NavigationCompleted(&g_navigation_completed_handler, &token);
    g_webview->add_NewWindowRequested(&g_new_window_handler, &token);
    g_webview->add_PermissionRequested(&g_permission_handler, &token);

    result = g_webview->Navigate(kEntryUrl);
    if (FAILED(result)) {
      FailAndQuit(L"Navigation request failed.");
    }
    return result;
  }
};

ControllerHandler g_controller_handler;

struct EnvironmentHandler final : ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler {
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** result) override {
    return QuerySingle(this, riid, IID_ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler, result);
  }
  ULONG STDMETHODCALLTYPE AddRef() override { return StableAddRef(); }
  ULONG STDMETHODCALLTYPE Release() override { return StableRelease(); }
  HRESULT STDMETHODCALLTYPE Invoke(HRESULT error_code, ICoreWebView2Environment* environment) override {
    if (FAILED(error_code) || environment == nullptr) {
      FailAndQuit(L"WebView2 environment creation failed.");
      return FAILED(error_code) ? error_code : E_FAIL;
    }
    return environment->CreateCoreWebView2Controller(g_window, &g_controller_handler);
  }
};

EnvironmentHandler g_environment_handler;

LRESULT CALLBACK WindowProc(HWND window, UINT message, WPARAM wparam, LPARAM lparam) noexcept {
  switch (message) {
    case WM_SIZE:
      ResizeController();
      return 0;
    case WM_DESTROY:
      PostQuitMessage(0);
      return 0;
    default:
      return DefWindowProcW(window, message, wparam, lparam);
  }
}

bool ResolvePath(const wchar_t* input, wchar_t* output, DWORD capacity) noexcept {
  if (input == nullptr || output == nullptr) {
    return false;
  }
  const DWORD length = GetFullPathNameW(input, capacity, output, nullptr);
  return length > 0 && length < capacity;
}

using CreateEnvironmentFn = HRESULT(STDAPICALLTYPE*)(
    PCWSTR, PCWSTR, ICoreWebView2EnvironmentOptions*,
    ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler*);

int Run() noexcept {
  int argument_count = 0;
  LPWSTR* arguments = CommandLineToArgvW(GetCommandLineW(), &argument_count);
  if (arguments == nullptr || argument_count != 3) {
    FailAndQuit(L"Usage: velox-host-cpp.exe <asset-root> <user-data-dir>");
    LocalFree(arguments);
    return 2;
  }
  const bool paths_ok = ResolvePath(arguments[1], g_asset_root, 32768) &&
                        ResolvePath(arguments[2], g_data_path, 32768);
  LocalFree(arguments);
  if (!paths_ok) {
    return 2;
  }

  if (FAILED(CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED))) {
    return 5;
  }

  WNDCLASSW window_class = {};
  window_class.lpfnWndProc = WindowProc;
  window_class.hInstance = GetModuleHandleW(nullptr);
  window_class.lpszClassName = kWindowClass;
  window_class.hCursor = LoadCursorW(nullptr, IDC_ARROW);
  if (RegisterClassW(&window_class) == 0 && GetLastError() != ERROR_CLASS_ALREADY_EXISTS) {
    CoUninitialize();
    return 5;
  }

  g_window = CreateWindowExW(0, kWindowClass, L"Velox C++23 Reference", WS_OVERLAPPEDWINDOW,
                             CW_USEDEFAULT, CW_USEDEFAULT, 720, 480, nullptr, nullptr,
                             GetModuleHandleW(nullptr), nullptr);
  if (g_window == nullptr) {
    CoUninitialize();
    return 5;
  }
  ShowWindow(g_window, SW_SHOW);

  HMODULE loader = LoadLibraryW(L"WebView2Loader.dll");
  if (loader == nullptr) {
    FailAndQuit(L"WebView2Loader.dll is unavailable.");
  } else {
    auto create_environment = reinterpret_cast<CreateEnvironmentFn>(
        GetProcAddress(loader, "CreateCoreWebView2EnvironmentWithOptions"));
    if (create_environment == nullptr ||
        FAILED(create_environment(nullptr, g_data_path, nullptr, &g_environment_handler))) {
      FailAndQuit(L"WebView2 environment request failed.");
    }
  }

  MSG message = {};
  while (GetMessageW(&message, nullptr, 0, 0) > 0) {
    TranslateMessage(&message);
    DispatchMessageW(&message);
  }

  if (g_controller != nullptr) {
    g_controller->Close();
    g_controller->Release();
    g_controller = nullptr;
  }
  if (g_webview != nullptr) {
    g_webview->Release();
    g_webview = nullptr;
  }
  if (loader != nullptr) {
    FreeLibrary(loader);
  }
  CoUninitialize();
  return static_cast<int>(message.wParam);
}

}  // namespace

extern "C" void wWinMainCRTStartup() noexcept {
  ExitProcess(static_cast<UINT>(Run()));
}
