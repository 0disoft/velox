# Velox fork notes

This directory is a narrow source fork of
`github.com/jchv/go-webview2@v0.0.0-20260205173254-56598839c808`.
The upstream MIT license is preserved in `LICENSE`.

Velox carries only the changes required by its Windows host boundary:

- use only the embedded WebView2 SDK loader, never an ambient DLL from the
  working directory or PATH, and retain initialization errors across calls;
- deny all WebView2 permission requests by default;
- expose virtual-host-to-folder mapping through the public wrapper;
- expose a synchronous benchmarkable web-resource response hook with balanced
  request, response, and response-stream COM ownership;
- validate WebMessage sources before dispatching bound callbacks;
- bound accepted WebMessages to 64 KiB and reject native conversion failures;
- deny untrusted top-level navigation and all frame navigation;
- deny popup and download events;
- add an opt-in `DisableReputationChecking` option, false by default, and
  require a `NavigationAllowed` callback plus denied frames, popups and
  downloads before querying `ICoreWebView2Settings8`;
- set `IsReputationCheckingRequired` false before the first navigation, release
  the queried interface, and keep the default when the runtime returns
  `E_NOINTERFACE`;
- allow an explicit fixed-runtime folder for missing-runtime conformance tests;
- expose the main WebView2 browser process ID for lifecycle measurement;
- expose a phase-only startup observer for benchmark instrumentation without
  changing WebView2 initialization decisions;
- expose a phase-only shutdown observer around handler removal, controller
  close, COM release, native window destruction, and message-loop exit;
- close and release the WebView2 controller, webview, and environment;
- balance the settings interface `AddRef` and `Release` methods and release each
  `GetSettings` reference after policy configuration;
- unregister native event handlers during shutdown;
- reject a zero native window before registering package-owned context;
- release a partially constructed browser before destroying a failed embed;
- process the native close loop before returning from settings-stage failures;
- discard queued binding responses after native window shutdown begins;
- handle ProcessFailed for main-browser exit so later user close can post native
  teardown without relying on a closed WebView; renderer/GPU/unknown failures
  do not bypass document consent;
- fail initialization when mandatory WebMessage or permission policies cannot
  be registered, without terminating the embedding process from the library;
- release queried `ICoreWebView2_3` interfaces; and
- remove window context after native window destruction.

## SmartScreen reputation setting

Microsoft documents `IsReputationCheckingRequired` as a SmartScreen switch
shared by every WebView using the same user-data folder. It defaults to true,
a false value applies on the next navigation or download, and another WebView
that leaves the default in place can reset the shared state to true.
SmartScreen is documented as help for reported phishing and malware sites and
download decisions, and as unnecessary overhead for app content. SDK
1.0.1722.45 places `GetIsReputationCheckingRequired` and
`PutIsReputationCheckingRequired` at slots 35 and 36 of the
`ICoreWebView2Settings8` vtable after Settings1-7; the fork uses IID
`9E6B0E8F-86AD-4E81-8147-A9B5EDB68650`.

`DisableReputationChecking` defaults to false. Enabling it requires
`NavigationAllowed` to be non-nil and `DenyFrames`, `DenyNewWindows` and
`DenyDownloads` to be true. The fork checks those fields; the callback itself
must restrict navigation to app-provided content. The call occurs before the
first navigation. A successful query releases the Settings8 pointer.
`E_NOINTERFACE` reports unsupported, leaves SmartScreen enabled and does not
fail construction. Any other HRESULT becomes an initialization error. The
initial settings reference is released on both success and failure.

The opt-out covers SmartScreen navigation and download reputation in the
app-specific profile. Local assets remain untrusted and are not tamper-proof.
Windows and Edge-wide SmartScreen, TLS validation, antivirus and Authenticode
warnings are unchanged. No separate subresource protection claim is made.
Remote navigation or downloads would require a policy review. See
[ICoreWebView2Settings8](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2settings8?view=webview2-1.0.4129.50)
and the
[IsSmartScreenRequired specification](https://github.com/MicrosoftEdge/WebView2Feedback/blob/main/specs/IsSmartScreenRequired.md).

Do not merge upstream changes mechanically. Review COM ownership, public API
changes, generated bindings, loader changes, and license notices before
updating the pinned source revision. Native callback objects share an explicit
`runtime.Pinner` and reference-counted owner; Go collector or pointer-rule
changes still require a fresh lifetime review before adoption.

The maintained fork requires Go 1.26.0 and uses `golang.org/x/sys v0.48.0`,
aligned with the Velox root module. This dependency refresh preserves the
upstream source revision and local COM/permission changes. The BSD-3-Clause
notice for x/sys is recorded in the root `THIRD_PARTY_NOTICES.md`.

The upstream x86 and ARM64 loader files remain checked in even though the first
supported target is Windows x64. Removing them saves only about 231 KiB from
the source checkout, does not reduce the x64 executable because build tags
already exclude them, and would make upstream review noisier. Revisit only if
repository or CI transfer measurements make that source-only cost material.

The native-only file-permission maintenance operation reads the current profile
through ICoreWebView2_13 and ICoreWebView2Profile4. It accepts only an origin
approved by FileSystemAccessAllowed, never grants access, and resets only an
existing FileReadWrite denial to Default after native user consent. Both async
completion handlers participate in the pinned callback owner lifetime (fifteen
handlers including ProcessFailed in the current source).
Destroy cancels the pending operation, releases its profile reference, and
suppresses late completion delivery. The public JavaScript IPC is unchanged.

The native constructor `NewWithOptionsAndError` distinguishes an early user
window close from initialization failure. `NewWithOptions` remains compatible
and still returns nil on either unsuccessful construction path. Only the
ordinary pre-consent user-close path marks cancellation; internal teardown
does not. A recorded Chromium initialization error takes precedence over that
marker. The Velox host treats the cancellation sentinel as a quiet exit 0;
actual initialization failures retain their error behavior.

The ProcessFailed handler is registered with a checked HRESULT and its own
event token, removed only after successful registration, and participates in
the existing pinning/refcount/QueryInterface contract. Only a successfully read
BrowserProcessExited kind marks the view unusable. The callback marks state;
it does not destroy the window inside COM or discard a healthy page on a timer.
The source opt-in native test uses a new private profile in a bounded subprocess,
terminates only its reported WebView2 browser PID after checking the executable,
and posts WM_CLOSE or SC_CLOSE after the real failure event. Repeat with
`VELOX_NATIVE_PROCESS_FAILURE=1 go test -v -run '^TestNativeBrowserFailureClose$' .`
from this fork (set the environment variable separately in PowerShell).
