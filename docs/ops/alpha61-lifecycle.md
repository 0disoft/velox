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
