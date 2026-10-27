# WebView2 112 Runtime Switch: 2026-10-28

Product commit `d8d0d43` handles a requested WebView2 runtime change during startup.
It checks the returned environment before creating a controller and retries while
the previous browser still supplies the wrong runtime. The transition stops when
the SDK supplies the selected version or the 30-second deadline expires. The older
fixture-side wait and its measurements are historical test evidence.

Current implementation:
[`runtime_transition_windows.go`](../../third_party/go-webview2/pkg/edge/runtime_transition_windows.go)
and [`chromium.go`](../../third_party/go-webview2/pkg/edge/chromium.go). The
current receipts are in `.cache/product-handoff-20261028/`.

## Current product behavior

- Each environment creation attempt resolves the selected folder's available
  runtime version. A retry resolves it again so an Evergreen update during the
  old browser's exit becomes the new target.
- `EnvironmentCompleted` reads the returned environment's
  `GetBrowserVersionString` before creating a controller and compares the actual
  version with the requested version.
- A mismatched environment is rejected without `AddRef`, without assignment to
  `e.environment`, and without `CreateCoreWebView2Controller`. That prevents a
  borrowed old environment from running an application page.
- A mismatch starts the transition deadline on the first mismatch and schedules
  a retry through the UI message pump every 100 ms. The total transition cap is
  30 seconds.
- The version read runs on the UI/COM thread. A COM completion can arrive inside
  `GetMessage`, so failures post a message to wake the pump.
- If the initial version lookup fails, startup keeps the previous SDK selection
  path. If a later lookup fails, the known target version and original deadline
  remain in effect; lookup failures do not silently accept an arbitrary runtime
  or extend the cap.
- A matching environment proceeds normally. Same-version startup adds no wait.
- The Win32 retry timer is removed on success, error, timeout and `Destroy`.
- The transition does not kill the browser, reset the profile, or change the
  data directory.

## Current verification

The uninstrumented product binary at
`.cache/product-handoff-20261028/host-product.exe` has SHA-256
`37cd9c547a2bf8b114b3ed0c5ddd547d4573b8abca396733f58da98a6fefa0e2`.
`product-111-to-112.log` passed:

| Measurement | Result |
| --- | --- |
| Previous browser exit after old host | 6.325921s |
| First next-host ready | 7.1288253s |
| Second next-host ready | 0.780588s |
| Old-host exit to first next-host ready | 7.1571575s |

The startup test now launches the next host immediately after each host exits.
It does not wait for the browser itself. Both next launches read the exact
Korean/CRLF IndexedDB draft, and every launch checks the profile with
`os.SameFile` plus an unchanged marker. The runner observes browser exit only in
cleanup, after the launch sequence.

`product-same-111.log` is a same-version product control. Its successive ready
times were 0.8867986s, 2.1562946s and 1.7201913s while a full test suite was
active. Those timings are not a performance benchmark. The control demonstrates
that the same-version branch does not add the runtime-transition wait.

`TestNativeRuntimeTransitionCleanup` passed timeout and cancellation cases with
the real SDK environment and a forced test version mismatch. The timeout case
used a shortened deadline. Both cases left timers at 0, callback references at
0 and the profile released. This test does not reproduce a real browser stuck
for the production 30-second cap.

The historical committed fixture wait and the
`TestUpgradeRuntimeChangeBoundary` helper have been removed. The current startup
test covers the product boundary by launching immediately.

Full validation then passed for both the product and the vendored fork:
`go test ./...`, `go test github.com/jchv/go-webview2/...`, `go vet ./...` and
`go vet github.com/jchv/go-webview2/...` used the isolated
`GOCACHE=.cache/runtime-switch-112-20261028/go-build-cache`. The first attempt
with the default Go cache failed with `not package main` and `bufio: buffer
full`; the isolated rerun succeeded, and the global cache was not deleted. The
fresh fork's full suite and `go test ./tests/hygiene -count=1` also passed.

Native fixed-runtime checks passed for 111.0.1661.62 and 112.0.1722.68. They
verified the selected executable path and version, the expected `Settings8`
value and the legacy reputation-option guard. A native cleanup rerun
(`.cache/product-handoff-20261028/native-cleanup.log`) passed both timeout and
cancellation: timers 0, callback references 0, profile released. The product host's
three early-window-close attempts also passed (`product-early-close.log`). Final
inspection found no fixed-runtime WebView2 process remaining under this checkout's
`.cache` directory.

## Historical cause trace

The following findings came from a test-only trace, not current product code.
They explain why the earlier launch did not reach ready.

Creating the environment while the old 111 browser was alive left the new host
using runtime 111 despite requesting the 112 folder. A trace build (SHA-256
`5e0ece8a...db71`, ignored overlay on `36c94ba` code, not shipping) logged:

- `environment-completed` and `controller-completed` returned `S_OK`; navigation
  completed with `success=1 status=0`.
- `BrowserVersionString` was 111.0.1661.62, and PID 39532 ran the 111 folder's
  `msedgewebview2.exe`.
- The witness reported `diagnostic-error:Unexpected-runtime-111.0.1661.62`, so
  its version assertion withheld the ready marker and the 15s watchdog failed.

A `TargetCompatibleBrowserVersion` attempt (`target-version.log`) behaved the
same: supplying the property did not force 112. Microsoft documents a
compatibility target whose actual binaries may differ, so verify the actual
environment version with
[`GetBrowserVersionString`](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment?view=webview2-1.0.3065.39#getbrowserversionstring),
and wait for `BrowserProcessExited` before changing runtime
([process events](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-related-events#updating-the-webview2-runtime)).

The requested folder and loader version describe intent; the actual runtime
comes from the environment and process. Initialization and navigation succeeded,
so this was not a 112 draft-damage or general 112 incompatibility finding.

## Historical test-only measurements

These values are preserved for context. They came from a fixture-side wait and
diagnostic overlays, not the current product implementation.
Their receipts remain in `.cache/runtime-switch-112-20261028/`.

The historical test commit `e9a0ee3` waited for the old fixture browser's process
handle to exit (up to 10s) before creating the next environment when the fixed
folder or expected version changed. Same-runtime relaunches stayed immediate.
Seven boundary cases covered same, case/clean, inherited, different,
default-version, fixed-to-default and unspecified inputs. That manual wait and
the boundary test are no longer in the current test.

`after-native.log` was the instrumented trace path: old browser exit wait
6.378001s, both new launches verified actual 112.0.1722.68 and the 112 image
(PIDs 384 and 28132), ready 0.6162158s and 0.5447423s, end-to-end handoff
7.0097536s. The Korean CRLF draft was read twice; directory identity and marker
held.

`after-plain-112.log` repeated it on the uninstrumented cached host (SHA-256
`5eec8cfd...77a8c`): exit wait 6.3771527s, first 112 ready 0.5832744s, second
0.5989189s, and old-host-exit-to-first-112-ready 6.9768804s. The first ready
alone excluded the wait and was not a speedup.

`same-111.log` kept the historical same-runtime branch immediate: first new 111
ready 7.043332s (old browser 6.3897639s), second 0.6991014s, with no
`runtime-change-browser-exit-wait` line. The earlier `go test ./tests/startup
./tests/hygiene`, `go vet ./tests/startup` and `git diff --check` receipts also
belong to that test-only state.

## Limits

The old host in the product run is the local pre-fix build with SHA-256
`3979158f...aedb4`, not an official release. Both sides still report
`0.5.10-alpha.68`; this is not an official release upgrade. The new product
binary is a local cached build.

No reduction of the old approximately 7-second first transition has been
established. The wait moves to product startup and the old browser still needs
about 6.3 seconds to release. A real 30-second stuck-browser timeout was not
reproduced.

This change adds no CLI, API, database, dependency or CI change. CI, physical IME,
installation, runtime downgrade, push and release were not performed. These are
local source-build fixtures with synthetic profiles, not release-upgrade evidence.

## Reproduction

```powershell
$env:VELOX_UPGRADE_PREVIOUS_HOST = '<pre-fix local host>'
$env:VELOX_BUILT_HOST = '<host-product.exe>'
$env:VELOX_UPGRADE_PREVIOUS_BROWSER_DIR = '<fixed 111 folder>'
$env:VELOX_BENCH_WEBVIEW2_BROWSER_DIR = '<fixed 112 folder>'
$env:VELOX_UPGRADE_PREVIOUS_VERSION = '111.0.1661.62'
$env:VELOX_UPGRADE_NEXT_VERSION = '112.0.1722.68'
go test ./tests/startup -run '^TestBuiltHostUpgradeHandoff$' -count=1 -v
```
