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
