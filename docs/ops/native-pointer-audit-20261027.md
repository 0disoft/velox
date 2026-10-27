# Native Pointer Audit: 2026-10-27

Local source follow-up on `main`: `d265926`, `c7173e6`, `024a678`.
Scope is the eight fork `unsafe.Pointer` vet warnings and related pointer
patterns, not a complete native-memory or security audit.

## Findings and changes

- `d265926`: one warning in `corewebview2.go:87`. A COM pointer retained as
  `uintptr` for deferred release lost Go pointer tracking. The existing
  `TestFilePermissionResetOnlyChangesMatchedDenialAndVerifies` aborted under
  `checkptr=2` in `releaseIUnknown`. The helper now accepts `unsafe.Pointer`
  and keeps it alive through `Release` with `runtime.KeepAlive`.
  Native `SHCreateMemStream` addresses remain integers: Windows
  `IUnknown_AtomicRelease` receives the address variable's pointer, releases
  the interface and clears the variable. Stream creation, release and repeated
  release passed on Windows.
- `c7173e6`: two `wndproc` warnings involved direct address casts for
  `WM_DPICHANGED` and `WM_GETMINMAXINFO`. The existing
  `TestDPIChangedAppliesSuggestedBounds` aborted under `checkptr=2` at the
  former `webview.go:380` cast. Fixed-size `ReadProcessMemory` and
  `WriteProcessMemory` copies now require success and the complete byte count.
  Tests passed for ignored addresses 0, 1 and maximum `uintptr`, and real
  `SendMessageW` DPI handling with negative coordinates and unchanged fields
  preserved.
- The same commit removes the one warning from converting an integer HWND to
  a Go memory pointer. **Go source compatibility changes:** the fork's
  `WebView.Window()` now returns `uintptr`, while upstream returns
  `unsafe.Pointer`. Repository callers and fake implementations were updated;
  external Go consumers must adapt their signatures and handle use.
- `024a678`: four warnings in three test files came from native UTF-16
  allocations. They were not reproduced as Go GC lifetime defects. The shared
  `assignTaskString` fixture uses the typed `SHStrDupW` output pointer and
  preserves the real COM getter's `CoTaskMemAlloc`/`CoTaskMemFree` contract.
  Duplicated URI allocation helpers were removed.

The two checkptr aborts used Go-owned fixtures. They demonstrate invalid Go
pointer-conversion patterns, not observed use-after-free of production
WebView2 native-heap objects. No warning suppression, integer bitcast workaround
or disabled validation was used.

## Evidence and limits

Receipts below are local files under `.cache/unsafe-pointer-20261027/`.

| Check | Result | Receipt |
| --- | --- | --- |
| Existing permission test before fix, `checkptr=2` | Fatal abort in `releaseIUnknown` | `before-checkptr.log` |
| Existing DPI test before fix, `checkptr=2` | Fatal abort in `wndproc` | `before-dpi-checkptr.log` |
| `go test -gcflags=all=-d=checkptr=2 github.com/jchv/go-webview2/... ./internal/webview2 -count=1` | Pass: four fork packages and one product package | `after-checkptr.log` |
| Combined fork/product vet | Pass: exit 0, empty output | `after-vet.log` |
| `go vet ./... github.com/jchv/go-webview2/...` | Pass: exit 0, empty output | `all-vet.log` |
| `go test ./... github.com/jchv/go-webview2/... -count=1 -timeout=5m` | Pass: exit 0; 48 packages ok (44 root, four fork), 10 no-test entries, zero failures | `all-tests.log` |
| Rebuilt-host `TestBuiltHostStartup`, explicit `VELOX_BUILT_HOST` | Pass: five of five subcases, 7.19s | `native-startup.log` |
| Immediate same-profile relaunch | Pass: three pairs | `native-lifecycle.json` |
| `TestNativeFilePermissionRecovery`, `VELOX_NATIVE_FILE_PERMISSION=1`, `checkptr=2` | Pass: real runtime, 3.19s | `native-file-permission.log` |

Native startup and lifecycle used the freshly built
`.cache/unsafe-pointer-20261027/velox-host.exe`. Lifecycle immediate-ready
times were 721.0482, 759.6263 and 1883.0687ms; first-browser exits after the
first host were 189.2022, 308.4921 and 1420.5848ms; profile release after the
immediate host was 258.2818, 263.6297 and 227.2007ms. These separate samples
do not establish speed equivalence or retention of the earlier 0.42s readiness.
The reason for the timing difference was not isolated in this pointer audit.

The separate opt-in permission run verified denial persistence across restart,
explicit reset and verification, reset persistence after another restart, and
continued rejection of another origin. Ordinary package passes alone do not
establish opt-in native coverage.

Older real runtimes, manual IME and hosted CI were not exercised. No push or
release was performed.

The eight warnings are resolved in the recorded vet runs. This does not prove
all pointer, API, database, permission or security risks are resolved. Velox web
app APIs, IPC, manifests, database formats, permissions, CI, dependencies and
the supported runtime floor are unchanged. The beta hold and release status
are unchanged; these are local source and test results.

## Reference contracts

[Go pointer rules](https://pkg.go.dev/unsafe#Pointer) explain that an integer
address does not retain Go pointer lifetime information.
[IUnknown_AtomicRelease](https://learn.microsoft.com/en-us/windows/win32/api/shlwapi/nf-shlwapi-iunknown_atomicrelease)
releases and clears a COM interface pointer and accepts an already-null value.
[SHStrDupW](https://learn.microsoft.com/en-us/windows/win32/api/shlwapi/nf-shlwapi-shstrdupw)
returns a task-allocated string that the caller frees with `CoTaskMemFree`.
