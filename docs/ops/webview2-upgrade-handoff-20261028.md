# WebView2 Upgrade Handoff: 2026-10-28

Follow-up to the 111 relaunch fix (`ef87a4d`): whether a newer host or an updater
can shorten the first launch when a browser started by an older host is still
alive. Product source and performance behavior are unchanged. Receipts are in
`.cache/upgrade-handoff-20261028/`.

## Measurements

Hosts are the old build (`7da29a2`, `vcs.modified=true`, SHA-256 `3979158f...aedb4`)
and the fixed build (SHA-256 `5eec8cfd...77a8c`). One physical profile, a distinct
phase URL per launch, and an exact Korean CRLF draft written by the old host then
read by both new launches. `os.SameFile` on the profile and an untouched marker held
after every launch.

| Fixture | Old browser exit after old host | First new ready | Ready after that exit | Second new ready |
| --- | --- | --- | --- | --- |
| Old host to fixed, both 111 | 6.4115972s | 6.98698s | 0.5819331s | 0.5496457s |
| Fixed to fixed, control | 0.0723988s | 0.4671022s | 0.400525s | 0.5252332s |
| Old 111 to 112, explicit exit wait | 6.4385909s | 0.6295184s | 0.6471938s | 0.5759082s |

The first two rows are `same-runtime-final.log` and `updated-host-control.log`. The
112 row is a diagnostic Go overlay (`wait_previous_exit_test.go`, not
committed) that waits for the old browser to exit before launching 112. Its
old-host-exit-to-next-ready is about 7.09s, so the wait moved, it did not shrink.
Microsoft states that the switch is a creation-time flag and cannot be changed at
runtime ([IsSmartScreenRequired specification](https://github.com/MicrosoftEdge/WebView2Feedback/blob/main/specs/IsSmartScreenRequired.md)),
and that configuration or runtime changes should wait for `BrowserProcessExited`
([process-related events](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-related-events#updating-the-webview2-runtime)).
A supported API to change the creation options of an already running browser was
not found.

## Failed runtime switch

Attempts to launch fixed 112 directly after the old 111 host, without the exit
wait, did not reach ready within 15s (`runtime-111-to-112.log`,
`runtime-111-to-112-detail.log`, `runtime-switch-cdp.log`). The owned fixture host
was force-stopped at the timeout and its synthetic profile was preserved. The
runtime-level cause was not isolated, and these runs do not establish that 112
is generally incompatible.

## Conclusion

No measured safe change shortens the first 7s wait. Keep the existing relaunch
fix and this native preservation regression. An updater could overlap an orderly
old-browser shutdown with installation work it already performs; no updater
integration, hidden prewarm or overlap benefit was performed or proven. The remaining
delay is a lingering lifetime/profile-release wait around the old browser.

## Verification

`479c50f` adds the opt-in `tests/startup/upgrade_handoff_windows_test.go`; no CLI,
API, database, dependency, CI runner or hygiene contract changed. The new case
skips without `VELOX_UPGRADE_PREVIOUS_HOST`. Only local opt-in runs (111 native,
112 after exit) plus `go test ./tests/startup ./tests/hygiene -count=1`,
`go vet ./tests/startup` and `git diff --check` were performed. The full Go suite
was not rerun because production code did not change. No CI, install, physical
IME, push or release was performed, and no user profile was touched or browser
killed.

## Reproduction

```powershell
$env:VELOX_UPGRADE_PREVIOUS_HOST = '<old velox-host.exe>'
$env:VELOX_BUILT_HOST = '<new velox-host.exe>'
$env:VELOX_UPGRADE_PREVIOUS_BROWSER_DIR = '<fixed 111 folder>'
$env:VELOX_BENCH_WEBVIEW2_BROWSER_DIR = '<next fixed folder>'
$env:VELOX_UPGRADE_PREVIOUS_VERSION = '111.0.1661.62'
$env:VELOX_UPGRADE_NEXT_VERSION = '111.0.1661.62'
go test ./tests/startup -run '^TestBuiltHostUpgradeHandoff$' -count=1 -v
```

## Limits

The old host reports `7da29a2` and `vcs.modified=true`; it is a local build, not a
released version. Both sides report `0.5.10-alpha.68`; this is not a newer application
release. Results are single local fixtures, not a general runtime or updater
claim.

Follow-up: [112 runtime switch trace](webview2-runtime-switch-112-20261028.md) shows the
15s watchdog failure came from the witness rejecting the unexpected 111 runtime.
