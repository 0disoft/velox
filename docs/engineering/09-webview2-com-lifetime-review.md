# Pure-Go WebView2 COM Lifetime Review

- Status: Callback retention and partial-initialization regression checks complete; candidate live validation pending
- Reviewed: 2026-09-08
- Scope: `third_party/go-webview2`, `internal/webview2`, and host shutdown paths
- Risk links: SEC-004 and R-003

## Decision

The bounded pure-Go adapter remains viable for the current Windows-only static
host. The review found three concrete lifetime defects and fixes them without
adding native capability or changing the public IPC contract.

The review does not claim general memory safety. Retained callbacks now have
an explicit pinning and reference-counted reachability contract. The historical
live validation below predates this change and does not validate the new
candidate or all supported runtimes.

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
| Native callbacks | `edge.Chromium` and the callback lifetime registry | all eleven callback objects, their vtables, and the owner are pinned before the loader receives a pointer; native `AddRef` retains the shared lifetime | `Destroy` drops the owner reference after native teardown; only the final native `Release` removes the Go root and unpins the objects |
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

## Retained Callback Contract

`callback_lifetime.go` keeps a Go-visible root independently of the native
pointer. `Embed` pins the owner, all eleven callback allocations, and each
native-readable vtable before publishing a handler address. Callback fields
containing Go interfaces and policy state are interpreted only by Go thunks;
WebView2 reads the vtable and calls its function addresses. Future fields that
native code traverses must be added to the pin inventory.

Every handler forwards native `AddRef` and `Release` into the same synchronized
allocation lifetime. One owner reference remains until teardown ends. Closing
the window is not permission to unpin a callback still held by native code.
The final release removes the registry entry and calls `Unpin`. A newly created
but never embedded browser has no registry entry or pins to leak. The local
fork now requires Go 1.21 for `runtime.Pinner`; Velox already requires Go 1.26.

Each callback's `QueryInterface` returns its own pointer only for `IUnknown`
or its declared handler IID, and acquires a reference before returning success.
Unsupported interfaces (including agility) fail with a cleared output pointer.
The native-entry tests check identity, reference acquisition, unsupported IIDs,
and missing output pointers for all eleven callbacks. IID values were
cross-checked against the [published WebView2 binding declarations](https://github.com/zzl/go-webview2/tree/main/wv2).

The design follows the [Go pinning contract](https://pkg.go.dev/runtime#Pinner)
and [COM reference-counting contract](https://learn.microsoft.com/en-us/windows/win32/com/implementing-reference-counting).
It deliberately shares lifetime across handlers, which can retain unused
handlers longer but cannot free a still-referenced handler early.

## Failure-Path Controls

- Failed HRESULTs use the signed 32-bit HRESULT interpretation, including on
  Windows x64. A successful status with an empty environment/controller is
  rejected.
- Initialization checks errors, completion, and the WebView pointer before
  injecting the bridge. A stopped message loop cannot become a false success.
- `Destroy` is idempotent and marks the browser closed before native teardown.
  Late environment callbacks cannot restart initialization; a late borrowed
  controller is closed without retaining it.
- All nine event registrations check their result and remember success
  separately from the token value. Failure can write an out parameter, and zero
  can be a valid token. Neither case is used as the registration-state signal.
- Accelerator registration passes the token's address, not a pointer to the
  local pointer variable, and checks HRESULT rather than Win32 last-error.

The fork suite injects each of the nine registration failures through native
callback-backed COM vtables, checks that only successful registrations are
removed, and checks no bridge injection on failure. Separate tests call all
eleven native AddRef/Release entry points, retain them across `Destroy` and a
forced GC, and require the final reference to remove the root. These are
controlled ABI fixtures, not fault injection into a real WebView2 process.

## Residual Risk and Unverified Paths

An unbalanced native reference can retain the registry entry indefinitely.
The binding must not unpin on a timeout to hide that leak. Failed handler
removal therefore still relies on controller teardown and native Release to
finish ownership. A native component violating COM reference-count rules is
outside the proof supplied by these tests.

Thread-affinity, the complete COM interface surface, and the supported WebView2
runtime matrix are not certified by this change. Record a process leak,
callback after final release, or unstable shutdown as a reopened SEC-004/R-003
finding. The candidate still needs a live startup check and hosted evidence
before a release or beta claim.

## Local Validation: 2026-09-08

Source revision `02003a0` (`0.5.10-alpha.39`) was checked on Windows amd64 with
installed WebView2 `152.0.4191.66`:

- The fork's complete unit suite passed through `velox_design_com_test` with
  dependency downloads disabled. The `pkg/edge` package itself has no unit tests;
  this pass does not validate every native callback graph.
- The root Go suite and vet passed, including runtime shutdown ordering and the
  distinction between accepted and newly dispatched IPC requests during close.
- A freshly built host passed `velox_design_lifecycle_test`: ten fresh-profile
  launches and ten immediate same-profile relaunches in 143.85 seconds. Every
  pair reached ready, exited both host and browser processes, and released the
  profile within the harness bounds.

The raw `velox.startup-lifecycle/v3` record is retained locally at
`.cache/design-lifecycle/evidence.json` with outcome `success`, ten successful
samples, and evidence level `controlled-local-observation`. It has no hosted
runner or public-release identity and is not a qualifying LLM trial, a supported
runtime matrix, registration-failure injection, or a general memory-safety proof.

This historical observation does not close the candidate's release-runner
gate. Retain live results for the new candidate revision before a channel
decision; do not relabel the alpha.39 record as evidence for changed code.
