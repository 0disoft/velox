# Profile Lock Diagnostic

## Local Observation: 2026-10-11

The observed lock belongs to WebView2's browser process after the Velox host
has exited, not to a still-running Velox host. This identifies the immediate
blocking process; it does not explain all internal browser shutdown work or
prove that every native reference has been released.

Tests used the unchanged GUI host from source
`d37801736aee7667855a444f398001249b9c1705`, SHA-256
`2af75667ecbc43a02345dfeaea304aa310b74c2fd52741788ec7f507ce5cdaf5`,
Go 1.27.1 Windows x64 and WebView2 154.0.4258.62. The diagnostic checkout was
`39a203c4c324f2c96601536d6dc0d83893d1cd92` plus the test-only probes.

Four serial fresh/immediate same-profile pairs counterbalanced two methods:
the existing regression's delete-while-browser-alive loop, and a control that
preserved the folder until both observed browser process handles signaled exit.
The latter used a non-destructive DELETE-access handle probe on the lockfile.
Opening failed with sharing violation while it was locked; the lockfile later
disappeared. A missing lockfile was not counted as successful access.

| Method | Immediate ready | Immediate host exit | Browser exit after host | Folder removed after host |
| --- | ---: | ---: | ---: | ---: |
| Delete first | 6,928.00 ms | 94.08 ms | 6,181.88 ms | 6,194.09 ms |
| Browser exit first | 6,941.25 ms | 149.85 ms | 9,369.34 ms | 9,599.52 ms |
| Browser exit first | 7,447.42 ms | 121.96 ms | 6,249.63 ms | 6,379.02 ms |
| Delete first | 7,051.01 ms | 199.35 ms | 6,187.61 ms | 6,241.69 ms |

Controller Close succeeded and the webview, controller and environment release
markers preceded host exit in all four samples. Immediate readiness followed
the first browser's observed exit by 671-862 ms. The legacy deletion loop
removed 158/160 of 159/161 profile files on its first failed attempt, leaving
the live browser's lockfile. It therefore changes the directory while measuring
its release. The control still had a 9.37-second browser-exit delay, so these
samples do not support blaming the deletion loop for the shutdown delay.

A separate unchanged-host pair queried Restart Manager for the exact lockfile
without using shutdown or restart APIs. It reported only PID 44192, named
Microsoft Edge WebView2, matching the immediate launch's observed browser PID.
The query took 208.98 ms; this is instrumented timing. That browser exited
6,194.80 ms after host exit and folder removal finished at 6,370.01 ms.
A sixth pair verified the final diagnostic's PID-match assertion: only
WebView2 PID 41988 was reported, matching the observed browser; it exited
6,107.83 ms after host exit and removal finished at 6,232.43 ms. The query
took 290.41 ms. No matching diagnostic browser processes remained afterward.

## Decision

Keep the earlier integration trial's 10-second failure in the record. It did
not reproduce in these six pairs. Browser shutdown variability explains the
observed lock lifetime, but the reason that the earlier trial exceeded ten
seconds is unresolved. No timer, flush, antivirus, COM leak or specific internal
browser task was identified as its cause.

Do not force-kill the browser, rotate user profiles, or make host exit wait
for browser termination. The production runtime is unchanged. Before the next
lifecycle-test change, prefer observing both browser exits before deleting the
disposable folder, while preserving the existing overall deadline and reporting
browser-exit failures separately from post-exit file-lock failures.

Microsoft's [user-data-folder guidance](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/user-data-folder#deleting-user-data-folders)
instructs waiting for browser processes to exit before deleting files still in
use after host closure. [RmGetList](https://learn.microsoft.com/en-us/windows/win32/api/restartmanager/nf-restartmanager-rmgetlist)
provides the registered-resource users used for the ownership observation.
Neither document specifies a universal six-second shutdown duration.

The new diagnostics are explicit opt-in Windows tests, not extra default
native launches or new CI jobs. Unit tests validate the non-destructive handle
probe and Restart Manager owner lookup. Local raw evidence stays ignored at
`.cache/profile-release-20261011/result.json` and `owner-result.json`.
The final assertion receipt is `owner-verified-result.json` in that directory.
No user app/profile/document, public API, DB schema, dependency, host binary,
version or CI workflow was changed; no push or publication occurred.

## Inspection Order Change: 2026-10-11

Ordinary built-host lifecycle and the JSON lifecycle evidence collector now
wait for the first and immediate browser process handles to signal exit before
attempting folder removal. One 10-second budget starts at release-check entry;
each wait receives only its remaining time. This does not grant ten seconds
per browser plus another ten seconds for removal. Removal does not start or
retry after its deadline; an OS removal call already in progress cannot be
canceled, but late completion fails the shared-budget check.

Failures preserve the existing `first-browser-exit`/`immediate-browser-exit`
phases with `BROWSER_EXIT_FAILED`, while post-exit removal uses `profile-release`
with `PROFILE_RELEASE_FAILED`. Already-observed exit timings are retained on
a later failure. Unconfirmed browser exits or failed launches preserve the
disposable profile, including the ordinary test's automatic cleanup path.

Focused regressions cover ordering, missing/failed exit observations, an actual
Windows sharing lock after simulated browser exits, deadline sharing, late
observations, zero remaining budget and guarded cleanup. Both actual native
paths passed on the unchanged host: ordinary lifecycle in 13.70 seconds, and
one JSON evidence pair in 14.18 seconds. In the evidence pair, the immediate
host exited in 94.26 ms; the browser exited 6,176.26 ms later, and folder removal
finished at 6,258.42 ms after host exit (82.16 ms after browser exit).
The existing summary tool accepted the record. Evidence is ignored at
`.cache/profile-release-order-20261011/evidence.json` and `summary.json`.

The v3 record shape, toolVersion 2, measured boundaries and failure vocabulary
are unchanged; this section documents the inspection-order difference.
The earlier intermittent failure remains historical evidence, not a fixed
runtime bug. No production host, public API, database, dependency, version,
workflow or runner changes, push or publication occurred. Full release and
hosted stress were not repeated for this test-only change.
