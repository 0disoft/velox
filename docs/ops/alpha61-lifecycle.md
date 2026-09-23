# Alpha.61 Public Lifecycle Check

## Artifact and Scope

On 2026-09-16 the public `v0.5.10-alpha.61` Windows x64 ZIP was downloaded
directly from GitHub Releases and checked against the independently recorded
publication digest. No local host rebuild substituted for the public binary.

- Release commit: `3fbe332e35cd262df47e3865e4b1f4e926c734c4`.
- ZIP SHA-256: `c082c90cd15116617fd0a29b0f11cbc5a2080019bd6a5a05103f8fadbac8b4c5`.
- Host SHA-256: `c8e00c98fa083274fd7be1833d8f1a5f836972ee7780fe8581654ce89afdfa72`.
- Measurement commit: `7174ad873418fe23fe80cbdf51622d028c925b11`.
- Environment: local Windows amd64, WebView2 `153.0.4234.32`, Go `1.26.4`.
- The startup tests and maintained WebView2 fork had no diff from the release tag.

This is local evidence, not a hosted-runner stress result or coverage of every
supported Windows/WebView2 version. Existing user apps and profiles were not
modified. All measurements used disposable test profiles.

## Public Binary Results

`TestStartupLifecycleEvidence` passed ten fresh/immediate same-profile pairs
(20 launches) in 128.64 seconds, with ten complete successful samples.
Every sampled host exited, both observed main browser process handles signaled
exit, and the test profile was removable within the configured deadlines.
This is not a census of all unrelated WebView2 processes on the machine.

| Measurement | Observed range |
| --- | --- |
| Fresh-profile ready | 460.37-606.17 ms |
| Immediate same-profile ready | 863.76-6942.01 ms |
| Host exit, both launches | 35.16-82.81 ms |
| Browser exit after host, both launches | 1354.90-6278.73 ms |
| Profile release after immediate host exit | 5781.89-6300.55 ms |

The run passed lifecycle correctness checks, but immediate relaunch and profile
release still have multi-second latency. Do not describe this as an instant
restart or as elimination of the previously observed shutdown delay.

## Separate Source Cancellation Results

`TestNativeInitializationCancellation` passed all six cases in 1.59 seconds:
three environment-completion cases and three controller-pending cases. Each
finished with zero callback references and a released profile. All three late
controller cases observed browser exit (153-166 ms). The environment-completion
cases do not claim an observed browser process exit or a delayed environment
callback after shutdown.

These tests exercised the tag-matching source fork, not cancellation inside
the downloaded public EXE. The public early-close probe below is separate
from these COM-stage tests. The 50-pair hosted alpha.61 stress run remains
unverified. This bounded run does not promote beta or change a runtime/API/database
contract.

Local raw evidence is retained under `.cache/public-alpha61-lifecycle-20260916/`:
`binding.json`, `lifecycle.json`, `lifecycle.log`, `cancellation.log`, and
`exit-codes.json`. The cache is not a distributed product artifact.

## Public EXE Early Close: Failed Normal-Exit Gate

A subsequent 2026-09-16 probe reused the exact host hash above. It observed the
owned process's first main window and sent its ordinary close request before
the benchmark readiness pipe connected. No ready marker arrived for any early
close. This bounds the observation to pre-readiness window closure; it does
not identify a specific internal COM initialization stage.

Three corrected-harness repetitions produced the same result:

- Close requests were accepted 46.32-61.72 ms after launch.
- All hosts exited without forced termination in 88.41-118.48 ms.
- All returned exit code 5 and `WebView2 Runtime is unavailable or initialization failed`.
- Immediate same-profile relaunches emitted `ready dom-2raf` and exited 0 in
  556.75-608.38 ms. All three disposable profiles were subsequently removable.

The normal-exit gate failed: a deliberate pre-ready close is reported as runtime
unavailability. `internal/webview2/runtime_windows.go` maps a nil constructor
result to `ErrRuntimeUnavailable`; the host maps that error to exit code 5.
The next fix must distinguish user cancellation from genuine initialization
failure, without converting real missing-runtime failures to success.

The initial harness waited for host exit before reading its readiness pipe;
its relaunches timed out and required cleanup. Concurrent reading removed that
measurement deadlock. Those timeouts are retained as harness failures, not
claimed as a reproduced public-host relaunch defect. Raw original and retry
evidence remains under `.cache/public61-early-close-20260916/`, with the corrected
run in `retry-3/`. No runtime fix or new release is included in this record.

## Development Alpha.62 Fix

The subsequent source change distinguishes native user cancellation from
constructor failure and maps only cancellation to a quiet host exit 0.
Recorded Chromium initialization errors take precedence; internal teardown
does not mark user cancellation. The old native constructor remains compatible.
No JavaScript IPC or database contract changes.

The permanent `TestBuiltHostStartup/early-user-close` regression failed against
the unchanged public alpha.61 executable with exit 5, then passed all three
early-close/relaunch/profile-release pairs against the local alpha.62 build.
It now runs in the existing consumer-evidence startup step without an extra
workflow. The complete native startup suite passed in 43.80 seconds, including
the missing-runtime error case, icon resources, GUI subsystem, lifecycle, and
security policies. Focused Go/version/hygiene and maintained-fork tests passed.

This fixes the local candidate, not the already published alpha.61 binary.
No new release or replacement of the installed File Notes executable occurred.
The ordinary same-profile relaunch still took 7.03 seconds in this run; the
shutdown latency and hosted stress gates remain separate.

## Published Alpha.62 Verification

Consumer CI run 35076577814 passed at source
`02c9acb5035014d9e29a0eb5881a3cf5310f5d6d`. Publication run 35079091056
and public-download verification run 35079337819 then passed for alpha.62.
A fresh local public-URL download matched the independent publication ZIP
SHA-256 `10137ca603c5ba7f765d58f9e93fc78683f328aebad77659fd63e01367265871`.
Its host SHA-256 was
`651a9d87d16eee5687f4a1072226e3f9209a6ece438c0672e6c30e6680037679`.

The unchanged regression passed three early-close/relaunch/profile-release
pairs on this downloaded EXE. The complete native startup suite passed in
43.64 seconds, including genuine missing-runtime errors, icon resources,
GUI subsystem, ordinary lifecycle and security policies. Raw output and digest
binding remain in `.cache/public-alpha62-20260916/`. These public-artifact
results supersede the alpha.61 early-close failure for the current release,
without deleting its historical failure evidence.

Ordinary immediate relaunch was 6.97 seconds and profile release took about
6.40 seconds. Extended hosted stress and beta promotion remain separate.
No installed user app, profile, or document was replaced during verification.

## Hosted Public Alpha.62 Stress: 2026-09-16

[Run 35081507786](https://github.com/0disoft/velox/actions/runs/35081507786),
attempt 1, passed at measurement commit
`214217112ae26bc110749cd4efc54aaf42e384b3`. The workflow downloaded the immutable
public alpha.62 ZIP and verified the ZIP and host hashes recorded above; it
did not build a substitute host. The runner was Windows 2025 amd64,
image `win25-vs2026` / `20260907.229.1`, with WebView2 `152.0.4191.66`.

All 50 fresh/immediate same-profile pairs completed successfully (100 launches)
in 612.109 seconds. Each sampled host exited, both observed main-browser
process handles signaled exit, and each disposable profile was removable
within the existing deadlines. The hosted schema and complete-sample gate
passed. This is not a census of all unrelated WebView2 processes.

| Measurement | Minimum | Maximum | p50 | p95 |
| --- | ---: | ---: | ---: | ---: |
| Fresh-profile ready | 497.18 ms | 2744.74 ms | 611.33 ms | 726.74 ms |
| Immediate same-profile ready | 538.27 ms | 6363.54 ms | 5962.89 ms | 6150.60 ms |
| Host exit, both launches | 59.15 ms | 86.05 ms | 66.37 ms | 77.92 ms |
| Browser exit after host, both launches | 111.15 ms | 6017.04 ms | 5658.23 ms | 5922.93 ms |
| Profile release after immediate host exit | 123.22 ms | 6091.30 ms | 5790.60 ms | 6003.04 ms |

Percentiles use nearest rank over 50 observations, or 100 for combined launch
rows. Passing shutdown deadlines does not resolve the roughly six-second
same-profile relaunch and profile-release latency. Initialization cancellation
was not part of these 50 pairs (`initializationCancellationTested: false`);
the separate public early-close evidence above remains the applicable check.
One hosted image/runtime combination does not prove all supported environments
or authorize beta promotion.

Artifact `alpha62-lifecycle-stress-35081507786-1` retains `release-binding.json`
and `lifecycle.json` for 90 days. Downloaded raw evidence and the independently
checked run binding remain under `.cache/alpha62-hosted-stress/35081507786-1/`.
The raw lifecycle JSON SHA-256 is
`550a4e2c12e2fa5f0cd003db77310a0bb2df311105cf5b9ed4ec81562cd317e9`.
The workflow remains manual-only and preserves success or failure artifacts.
No runtime, public API, DB, release version, user profile or installed EXE was
changed by this stress task; the existing unsigned alpha.62 release is unchanged.

## Current-Source User Close Diagnostic: 2026-09-23

A local production-style host built from source at `4cdfb0868166b28b470985e0b58cba150a746ea7`
with Go `1.26.4` had SHA-256
`a23f086ef590b75595b09a0376ef4a68d92f9ae7fe1b9a4a6a4deeb5627e30eb`.
The installed Evergreen WebView2 Runtime was `153.0.4234.48`. This is current
source with a different Go build from the public alpha.62 host, not a new
public-artifact verification. Tests used disposable profiles under `.cache`;
no installed app or user profile was opened.

`velox_design_lifecycle_test` passed ten fresh/immediate same-profile pairs.
Nearest-rank p50 was 6,927 ms from second-host start to ready, 6,361 ms from
second-host start to first-browser exit, and 6,707 ms between the second
environment-created and controller-created markers. All ten second launches
reached ready after the first browser exited; one pair was much faster than
the other nine. The first host's shutdown request to run-loop exit had p50
60 ms. The local JSON SHA-256 is
`b258ab04935450955f5bd2e5eecff4d19d08905b9c2f01ebed32df2f13d8720b`.

One temporary ready-window close diagnostic exercised the normal `WM_CLOSE`
path instead of the benchmark's close-immediately-after-ready hook. In one
completed run, the host exited 153 ms after the close request, the observed
browser process exited 6,402 ms after host exit, and the disposable profile
was removable 108 ms later. That full `velox_startup_smoke` pass took 52.39 s.
An earlier attempt timed out at the intent's 60-second limit before the
diagnostic finished. On a subsequent repetition, the host close completed but
the browser-exit observation did not finish within 10 s; the 60-second smoke
then timed out before its remaining cases completed. No test profile or owned
host process remained afterward. These are diagnostic and runner failures,
not evidence that the host failed to close. The temporary subtest was removed
from the default smoke rather than making that 60-second gate timing-sensitive.
After removing it, the unchanged startup smoke passed in 47.39 s: first
browser exit after host exit was 6.412 s, and the immediate same-profile
launch reached ready in 7.120 s.

The completed run shows that a browser-process delay can occur after ordinary
ready-window closure, and the repetition shows that a fixed six-second duration
cannot be assumed. Neither run directly measures a second launch after manual
close or explains why WebView2 retains the browser process. Microsoft's
[user-data-folder guidance](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/user-data-folder)
notes that browser processes can keep profile files in use after the host
closes; it does not establish a universal six-second duration. No profile
rotation, forced browser termination, teardown reordering, release, or beta
promotion follows from this diagnostic.
