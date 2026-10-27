# WebView2 Upgrade Handoff: 2026-10-28

Product commit `d8d0d43` adds the runtime transition guard to the actual host.
Before creating a controller it checks the environment's actual runtime. A
mismatch retries through the UI message pump until the SDK supplies the selected
version, normally after the previous browser exits, with a 30-second cap.
The product keeps the same profile and does not terminate browsers or reset data.
Same-version startup adds no transition wait.

Implementation, current verification commands and failure-path limits are in
[112 runtime switch](webview2-runtime-switch-112-20261028.md). Earlier fixture-side
waits below are historical evidence; the current test launches immediately after
each host exits and relies on the product guard.

## Current product evidence

The uninstrumented local host has SHA-256
`37cd9c547a2bf8b114b3ed0c5ddd547d4573b8abca396733f58da98a6fefa0e2`.
Receipts are in `.cache/product-handoff-20261028/`.

| Measurement | Result |
| --- | --- |
| Previous 111 browser exit after old host | 6.325921s |
| First 112 host ready, including product wait | 7.1288253s |
| Second 112 host ready | 0.780588s |
| Old-host exit to first 112 ready | 7.1571575s |

`product-111-to-112.log` passed without any fixture-side wait. Both new launches
required actual 112.0.1722.68 and read the exact Korean/CRLF IndexedDB draft.
Profile directory identity (`os.SameFile`) and the untouched marker held after
every launch. The same-111 product control also passed; it ran under concurrent
full-suite load and is not a performance benchmark.

Full Go product/fork tests, vet, hygiene and native fixed 111/112 checks passed.
Native transition timeout (shortened test deadline) and cancellation released all
callback references, the timer and the profile. Three product-host early-close
attempts passed. The final fixed-runtime browser count under `.cache` was zero.

The first approximately 7-second transition is not shorter: the old browser still
needs about 6.3 seconds to exit. An updater could overlap orderly shutdown with
installation work, but no updater integration or overlap benefit was tested.

## Historical test-only measurements

Before the product guard, these local builds used the old host (`7da29a2`,
`vcs.modified=true`, SHA-256 `3979158f...aedb4`) and the fixed host (SHA-256
`5eec8cfd...77a8c`). Receipts remain in `.cache/upgrade-handoff-20261028/`.

| Fixture | Old browser exit after old host | First new ready | Ready after that exit | Second new ready |
| --- | --- | --- | --- | --- |
| Old host to fixed, both 111 | 6.4115972s | 6.98698s | 0.5819331s | 0.5496457s |
| Fixed to fixed, control | 0.0723988s | 0.4671022s | 0.400525s | 0.5252332s |
| Old 111 to 112, explicit exit wait | 6.4385909s | 0.6295184s | 0.6471938s | 0.5759082s |

The first two rows are `same-runtime-final.log` and `updated-host-control.log`.
The 112 row used an uncommitted diagnostic overlay that waited for browser exit
before launching the new host. Its end-to-end handoff was about 7.09s: the wait
moved rather than shrinking. `479c50f` added the preservation fixture; `e9a0ee3`
added a fixture-side wait and seven boundary cases. That wait and its boundary
helper are now removed.

Without either wait or product guard, direct 111-to-112 attempts failed the 15s
ready watchdog (`runtime-111-to-112.log`, `runtime-111-to-112-detail.log`,
`runtime-switch-cdp.log`). The owned fixture host was stopped and its synthetic
profile preserved. The later [runtime trace](webview2-runtime-switch-112-20261028.md#historical-cause-trace)
identified an actual 111 environment despite the requested 112 folder. The page's
runtime assertion withheld ready; this was not a general 112 incompatibility or
draft-damage finding.

Microsoft describes the reputation option as a creation-time flag
([specification](https://github.com/MicrosoftEdge/WebView2Feedback/blob/main/specs/IsSmartScreenRequired.md))
and recommends waiting for browser exit before changing runtime
([process events](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-related-events#updating-the-webview2-runtime)).

## Limits and reproduction

The previous host is a local pre-fix build, not an official release. Both sides
report `0.5.10-alpha.68`; these are synthetic-profile source-build fixtures, not
an official application upgrade. No actual 30-second stuck-browser timeout,
runtime downgrade, physical IME, installer, CI, push or release was tested.
CLI/API, database, dependency and CI contracts are unchanged.

```powershell
$env:VELOX_UPGRADE_PREVIOUS_HOST = '<pre-fix local host>'
$env:VELOX_BUILT_HOST = '<host-product.exe>'
$env:VELOX_UPGRADE_PREVIOUS_BROWSER_DIR = '<fixed 111 folder>'
$env:VELOX_BENCH_WEBVIEW2_BROWSER_DIR = '<fixed 112 folder>'
$env:VELOX_UPGRADE_PREVIOUS_VERSION = '111.0.1661.62'
$env:VELOX_UPGRADE_NEXT_VERSION = '112.0.1722.68'
go test ./tests/startup -run '^TestBuiltHostUpgradeHandoff$' -count=1 -v
```
