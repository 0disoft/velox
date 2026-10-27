# WebView2 112 Runtime Switch: 2026-10-28

Follow-up to [upgrade handoff](webview2-upgrade-handoff-20261028.md): why a direct
111-to-112 launch failed to reach ready. The cause is established; product code is
unchanged. Receipts are in `.cache/runtime-switch-112-20261028/`.

## Cause

Creating the environment while the old 111 browser is alive leaves the new host
using runtime 111 despite requesting the 112 folder. A trace build (SHA-256 `5e0ece8a...db71`, ignored
overlay on `36c94ba` code, not shipping) logged:

- `environment-completed` and `controller-completed` returned `S_OK`; navigation
  completed with `success=1 status=0`.
- `BrowserVersionString` was 111.0.1661.62, and PID 39532 ran the 111 folder's
  `msedgewebview2.exe`.
- The witness reported `diagnostic-error:Unexpected-runtime-111.0.1661.62`, so its
  version assertion withheld the ready marker and the 15s watchdog failed.

A `TargetCompatibleBrowserVersion` attempt (`target-version.log`) behaved the same:
supplying the property did not force 112. Microsoft documents a compatibility
target whose actual binaries may differ, so verify `GetBrowserVersionString`
([environment options](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environmentoptions?view=webview2-1.0.3065.39#get_targetcompatiblebrowserversion)),
and wait for `BrowserProcessExited` before changing runtime
([process events](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-related-events#updating-the-webview2-runtime)).

The requested folder and loader version describe intent; the actual runtime comes
from the environment and process. Initialization and navigation succeeded, so this
is not a 112 draft-damage or general 112 incompatibility finding.

## Change

Tests only (`tests/startup/upgrade_handoff_windows_test.go`): when the fixed folder
or expected version changes, observe the old fixture browser's process handle exit
(up to 10s) before creating the next environment. Same-runtime relaunches stay
immediate. The observed exit timestamp is reused by cleanup, and
`handoff-ready-after-old-host-exit` records end-to-end time so the wait cannot be
omitted. Seven boundary cases cover same, case/clean, inherited, different,
default-version, fixed-to-default and unspecified inputs.

## Verification

`after-native.log` is the instrumented trace path: old browser exit wait 6.378001s,
both new launches verified actual 112.0.1722.68 and the 112 image (PIDs 384 and
28132), ready 0.6162158s and 0.5447423s, end-to-end handoff 7.0097536s. The Korean
CRLF draft was read twice; directory identity and marker held.

`after-plain-112.log` repeats it on the uninstrumented cached host (SHA-256
`5eec8cfd...77a8c`): exit wait 6.3771527s, first 112 ready 0.5832744s, second
0.5989189s, and old-host-exit-to-first-112-ready 6.9768804s, so the first ready
alone excludes the wait and is not a speedup. `same-111.log` keeps the same-runtime
branch immediate: first new 111 ready 7.043332s (old browser 6.3897639s), second
0.6991014s, with no `runtime-change-browser-exit-wait` line.

## Limits

No product source, CLI, API, database, dependency or CI change; no automatic
handoff, browser kill or profile reset. The old host is the pre-fix build
(SHA-256 `3979158f...aedb4`, metadata `7da29a2` with `vcs.modified=true`), not a
release. Both report `0.5.10-alpha.68`; this is not an official release upgrade.
The trace host carries native IPC and log-only diagnostics and is not shipping
code; the plain and instrumented runs are separate receipts. The seven boundary
cases, `go test ./tests/startup ./tests/hygiene -count=1`, `go vet ./tests/startup`
and `git diff --check` passed; fixed-runtime browser remaining was 0. The full Go
suite was not rerun because product code did not change. CI, install, physical
IME, push and release were not performed.

## Reproduction

```powershell
$env:VELOX_UPGRADE_PREVIOUS_HOST = '<pre-fix host>'
$env:VELOX_BUILT_HOST = '<fixed host>'
$env:VELOX_UPGRADE_PREVIOUS_BROWSER_DIR = '<fixed 111 folder>'
$env:VELOX_BENCH_WEBVIEW2_BROWSER_DIR = '<fixed 112 folder>'
$env:VELOX_UPGRADE_PREVIOUS_VERSION = '111.0.1661.62'
$env:VELOX_UPGRADE_NEXT_VERSION = '112.0.1722.68'
go test ./tests/startup -run '^TestBuiltHostUpgradeHandoff$' -count=1 -v
go test ./tests/startup -run '^TestUpgradeRuntimeChangeBoundary$' -count=1 -v
```
