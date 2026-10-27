# WebView2 111 Relaunch Stall: 2026-10-27

Local `0.5.10-alpha.68` host, fixed runtime 111.0.1661.62, a dedicated app profile and
same-profile immediate relaunch. Receipts live in `.cache/relaunch-111-20261027/`.
Nothing here was pushed, tagged, installed, published or run on hosted CI; machine-wide
SmartScreen, the registry and the installed Evergreen runtime were unchanged.

## Experiment

Two repetitions per phase on 111 with the unchanged host (`baseline.json`,
`legacy-smartscreen-off.json`, `baseline-return.json`):

| Phase | Immediate ready (ms) | First browser exit (ms) |
| --- | --- | --- |
| Default, before | 7047.13; 7279.71 | 6240.75; 6321.70 |
| `--disable-features=msSmartScreenProtection` | 593.42; 539.77 | 111.29; 100.63 |
| Default, restored | 6704.37; 7136.96 | 6112.86; 6229.44 |

The switch came only from the diagnostic harness environment. `host.Close` measured
65.3ms, 73.3ms and 71.5ms in `startup-111.log`, and 76.3-108.9ms in the final run. The
multi-second wait sits between `environment-created` and `controller-created` on the
second host (6413.8-6906.8ms) while the first browser exits, so the host destructor is
not the six-second wait.

## Cause and fix

111.0.1661.62 returns `E_NOINTERFACE` for `ICoreWebView2Settings8`, so
`DisableReputationChecking` fell through its runtime-settings path and changed nothing
(`loader-111.0.1661.62.log`). Microsoft's
[IsSmartScreenRequired specification](https://github.com/MicrosoftEdge/WebView2Feedback/blob/main/specs/IsSmartScreenRequired.md)
documents `--disable-features=msSmartScreenProtection` for runtimes where that property
is unavailable. The switch at browser creation removed the stall and restoring the
default brought it back; this receipt supports that causal contrast, not a
protocol-level network diagnosis.

`b2523bf` adds `webviewloader.GetAvailableVersion(folder)`, resolving the version
through the same embedded loader and folder used for environment creation and making
the native witness assert it. `ef87a4d` adds a fork-owned
`ICoreWebView2EnvironmentOptions` object, passed to
`CreateCoreWebView2EnvironmentWithOptions` only when `DisableReputationChecking` is
true, `NavigationAllowed` is non-nil, `DenyFrames`, `DenyNewWindows` and
`DenyDownloads` are true, and the selected version is 92.0.0.0 through 111.x.x.x.
`AdditionalBrowserArguments` carries the switch and `TargetCompatibleBrowserVersion`
the version. An unknown or failed query leaves the flag off; 112+ keeps Settings8. The
object has three `IUnknown` slots plus four SDK 1.0.1722.45 get/put pairs, `HRESULT`
returns, a four-byte `BOOL`, and `SHStrDupW` task-memory strings; it shares the
existing pinner and reference-counted owner. No environment variable is mutated. The
legacy flag is fixed when the browser starts and has no runtime setter, so it cannot
be turned back on while that browser survives. `IsReputationCheckingRequired` applies
from the next navigation and another WebView that leaves the default can reset the
shared state. Velox uses either path only with the app-specific dedicated profile.

## Final results

Three post-fix 111 samples (`lifecycle-final-111.0.1661.62.json`): immediate ready
632.10, 603.68 and 567.47ms; first browser exit 94.15, 130.83 and 118.39ms; profile
removal 95.69, 104.29 and 137.22ms. `timing-summary.json` compares four baseline
samples with these three: median 7092.0462ms to 603.6787ms, a 91.49% reduction.

| Runtime | Samples | Immediate ready | First browser exit | Lifecycle |
| --- | --- | --- | --- | --- |
| 111.0.1661.62 | 3 | 603.68ms median | 118.39ms median | pass |
| 92.0.902.73 | 1 | 966.29ms | 121.24ms | pass |
| 112.0.1722.68 | 1 | 567.51ms | 53.28ms | pass |
| 155.0.4283.62 | 1 | 584.78ms | 109.89ms | pass |

`final-matrix.json` records exit 0 for draft, startup and lifecycle on all four. 111 and 92
passed all five `TestBuiltHostStartup` subcases; 112 and 155 passed lifecycle and security
policy. The draft smoke passed write, restore-clear and cleared-relaunch on each version
(`draft-final-*.log`). The native witness passed for 92, 111 and 112 with the actual path,
version and expected legacy/Settings8 choice (`options-native-*.log`); its deferred check
also fails unless the browser exits and the callback owner reaches zero references. Fork
packages passed in `fork-final.log`, and `vet-final.log` is empty at exit 0.

## Boundary and limits

Global SmartScreen, registry, installed Evergreen, TLS, antivirus and Authenticode are
unchanged. No user profile was deleted or moved; the product adds no browser termination.
`GetAvailableVersion` and the Chromium field are additive Go API changes; CLI and IPC
signatures, schema, dependencies and CI are unchanged.

Only 92.0.902.73 was tested; the 92.0.902.49 floor was not run directly, and the
patch.73 result is one build. The saved native logs do not record the `-d=checkptr=2`
setting, so treat that harness flag as separately reported. Behavior on every legacy
patch is not guaranteed. Physical keyboard and IME input were not exercised; draft
evidence is synthetic CDP input. The repository-wide `go test ./...` finished with
exit 0: 48 packages ok, no failures (`all-tests-result.json`).
`go test ./tests/hygiene -count=1` and `git diff --check` also passed.
In one mixed old-to-new host fixture (`mixed-upgrade.log`), the first same-profile
handoff took 7.0728s because the browser started by the old host stayed alive for 6.376s.

## Artifacts

The final host is `velox-host-final.exe` (SHA-256
`5eec8cfd84ab7cd342eb2ce1d850a3a6aa7fc6ccdcbfb4863db3f9c6fee77a8c`, read from the file),
a clean build of `ef87a4d`. The local bundle reports `0.5.10-alpha.68` and is not a
published release; `release/velox-windows-x64.zip` is 4,925,586 bytes with SHA-256
`92e14d672db87162ae49da99dbca4d81a51ad1b1c182747032f11a1f3328be4f`.

Follow-up: [upgrade handoff evidence](webview2-upgrade-handoff-20261028.md) records
the remaining first-launch wait and profile-preservation checks.
