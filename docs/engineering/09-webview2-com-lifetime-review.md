# Pure-Go WebView2 COM Lifetime Review

- Status: Source review complete; live Windows stress remains required before beta
- Reviewed: 2026-09-01
- Scope: `third_party/go-webview2`, `internal/webview2`, and host shutdown paths
- Risk links: SEC-004 and R-003

## Decision

The bounded pure-Go adapter remains viable for the current Windows-only static
host. The review found three concrete lifetime defects and fixes them without
adding native capability or changing the public IPC contract.

The review does not claim general memory safety. Native callbacks still retain
Go object addresses without an explicit `runtime.Pinner` contract, and live
repeated startup and shutdown under the supported WebView2 runtime remains a
beta gate.

## Ownership and Release Map

| Resource | Owner | Acquisition | Release or invalidation path |
| --- | --- | --- | --- |
| Win32 `HWND` | `webview` | `CreateWindowExW` | `WM_CLOSE` calls browser teardown, `DestroyWindow` emits `WM_DESTROY`, and the message loop exits through `WM_QUIT`; a zero handle is now rejected before registration |
| Window context entry | package `windowContext` map | immediately after a nonzero `HWND` is created | deleted by `WM_DESTROY`; failed embed destroys the native window synchronously |
| WebView2 environment | `edge.Chromium` | environment completion callback takes an explicit native `AddRef` | `Chromium.Destroy` releases and clears it after event-handler removal and controller teardown |
| WebView2 controller | `edge.Chromium` | controller completion callback takes an explicit native `AddRef` | `Close`, then `Release`, then clear during `Chromium.Destroy` |
| Core WebView2 interface | `edge.Chromium` | `GetCoreWebView2` returns the retained interface | released and cleared before controller and environment release |
| Settings interface | `NewWithOptions` configuration step | `GetSettings` returns a COM interface reference | `configureSettings` now defers exactly one `Release` on success and every failure path; the wrapper now supplies balanced `AddRef` and `Release` calls with the interface pointer |
| Queried versioned interfaces | individual method scope | `GetICoreWebView2_3` and `GetICoreWebView2_4` | released with scoped `defer` or explicit release before return |
| Native event callbacks | `edge.Chromium` fields | Go callback objects are registered with WebView2 and tokens are retained | registered handlers are removed before WebView2 release; Go fields remain reachable for the Chromium lifetime |
| Web resource request | `WebResourceRequested` callback scope | WebView2 returns the request interface | released with `defer` after callback handling |
| Web resource response and backing stream | `CreateWebResourceResponse` call scope | response and optional `SHCreateMemStream` result | response is released after `PutResponse`; stream is released through `releaseIUnknown` after response creation |
| Bound Go callbacks and queued responses | `webview.bindings` and `dispatchq` | `Bind` and synchronous WebMessage dispatch | dispatcher closes before native destroy; queued JavaScript responses re-check `closing` and are discarded after close begins |

## Confirmed Defects and Fixes

### COM-001: Settings references were leaked

`GetSettings` returned an `ICoreWebViewSettings` reference, but the wrapper had
no `Release` method and the initialization path never released it. The binding
now exposes balanced `AddRef` and `Release` methods, passes the interface pointer
to both calls, and releases the settings reference exactly once after applying
context-menu and developer-tools policy. Unit tests cover success and both
configuration failure positions.

### COM-002: Settings-stage failures queued close without pumping it

When settings lookup or policy application failed after the native window was
created, `NewWithOptions` posted `WM_CLOSE` and returned `nil`. No caller could
then enter `Run`, so window destruction and COM release could remain queued.
The failure path now performs `Destroy` followed by `Run`, guaranteeing that the
native close sequence is processed before the constructor returns. A focused
sequence test guards this contract.

### COM-003: Partial embed failure did not have one unconditional cleanup path

`CreateWithOptions` registered the window context without first rejecting a
zero `HWND`, and an `Embed` failure destroyed the window without explicitly
releasing a partially constructed browser. It now rejects a zero handle,
invokes browser teardown on every failed embed, and then destroys the native
window. A focused test verifies partial browser teardown without requiring a
live WebView2 runtime.

## Existing Lifecycle Controls Retained

`internal/webview2.Runtime.Close` remains protected by `sync.Once`, closes the
IPC dispatcher before native teardown, and gives a bound response two dispatch
turns to drain before destruction. Its existing tests cover repeated close and
destroy ordering. The fork's existing callback test covers a response queued
before close and confirms that it is not evaluated after close begins.

## Residual Risk and Unverified Paths

The callback objects passed to native code remain Go heap objects whose
addresses are converted through `unsafe.Pointer`. The current Go collector does
not move them, but the binding does not explicitly pin them. A future compacting
collector or changed cgo/syscall pointer rule requires either `runtime.Pinner`
coverage for every retained callback graph or replacement bindings with an
explicit native allocation strategy.

Event-registration teardown still relies on WebView2 tolerating removal calls
for tokens whose registration may have failed during partial initialization.
The source review found no observed failure from that behavior, but it remains a
candidate for registration-state tracking if live fault injection exposes one.

This change was reviewed without a local Windows desktop or installed WebView2
runtime. Before beta promotion, run the fork unit tests, root lifecycle tests,
and repeated startup and shutdown stress on the supported Windows runner. Record
any process leak, callback after release, thread-affinity violation, or unstable
shutdown phase as a reopened SEC-004/R-003 finding.
