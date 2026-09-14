# Alpha.54 Readiness Follow-Up

This record separates local source tests, visual observations and candidate
distribution checks. None implies beta admission or public release.

## 1. Initialization Cancellation

Source `b9d501f93b5e690a3528560c8d6b7154409c0b0e`, local Windows amd64,
Go `go1.26.4`, WebView2 `152.0.4191.66`.

One invocation of `TestNativeInitializationCancellation` passed all six cases
in 7.22 seconds: three environment-completion cancellations and three
controller-pending cancellations. All callback reference counts reached zero
and all private profiles were released. Late controller browser exits were
observed after 5,465 ms, 177 ms and 170 ms, with no process-wait errors.
Environment-completion cases do not claim browser-exit observation because
they do not schedule a controller.

The earlier intermittent failure did not reproduce. The slow first browser
exit and historical failure remain recorded; this passing run is not proof
that their cause was fixed. No runtime change or repeated run was made to
obtain a pass. Hosted verification is recorded separately below.

## 2. DPI and Visual Checks

The focused DPI/host, fork window/COM, version-fixture and hygiene checks
passed on alpha.54. These cover DPI setup and scaling behavior in tests;
they do not substitute for observed 125%/150% display scaling, text sharpness
or movement between differently scaled monitors.

The Windows UI helper could not obtain a targetable Display Settings window
after a launch attempt and refreshed window inventory. The user was asked to
open Display Settings. No display scaling was changed, no screenshot-based
sharpness claim is made, and the physical display matrix remains pending.
Candidate preparation may continue, but this pending check is retained in
the release decision rather than converted into a pass.

## 3. Candidate and Release Decision

Candidate source: `f181ace4c3e44f9ba652fac0015ea7b957ea85a7`. The changes since
the alpha.54 runtime commit are readiness documentation only.

The local unsigned Windows x64 ZIP was assembled and its checksum, SPDX and
unsigned provenance sidecars validated. Archive size: 3,605,116 bytes;
SHA-256: `f30d92f50728fbcce5a86a2b72cc06500e1d0b4bbd3b0d4dc94827207c0b26bc`.
The ZIP was extracted into a new private directory and its manifest-bound
host passed fresh/immediate startup, security policy and profile release in
22.53 seconds. Immediate readiness was 7.70 seconds; latency is not fixed.
Evidence directory: `zip54-7287a1ea-1e4a-45f6-b59d-6a6c6330e908`.

File Notes app/model tests passed all 11 cases, covering saved baselines,
edits during a write, picker cancellation, denied writes, stream failure,
restoration and Unicode state. They use browser API doubles, not real user
consent. The candidate CLI's File Notes smoke passed validation, doctor,
two identical application builds, archive inspection, packaged startup and
source startup. File Notes archive SHA-256:
`ac55e00772f19cc06beff9ea66fe6b2c4bff7168dd450b6dd5c9db025f282977`.

Two native development-mode HTML/CSS/JS reloads passed with
`Page.reload(ignoreCache: false)`, retaining the same origin and recording
successful process cleanup. Result: `normal-reload-1789376400669/result.json`,
2026-09-14 09:00:00-09:00:01 UTC. The inspected release-host SHA-256 was
`2b89c6e49023578140c573c74a7c1b01053556a6ef04115fb1a4b82f523a0373`.

### Hosted Windows

Both workflows completed successfully for exact source
`f181ace4c3e44f9ba652fac0015ea7b957ea85a7`:

- [Native initialization cancellation, run 34825059428](https://github.com/0disoft/velox/actions/runs/34825059428).
  The downloaded binding/result artifacts matched the source SHA, run ID,
  six requested cases and six passes. Environment: `win25-vs2026`, image
  `20260907.229.1`, Go `go1.26.8`, WebView2 `152.0.4191.66`, amd64.
  This is source-fork evidence, not a public release or late-environment
  callback test. Artifact directory: `native-cancellation-34825059428`.
- [Consumer evidence, run 34825107832](https://github.com/0disoft/velox/actions/runs/34825107832).
  The explicit quick workflow succeeded; profile comparison, history and
  fuzz campaigns were disabled. This verifies the workflow's own build,
  not byte identity with the locally built ZIP. The optional Actions warning
  monitor was skipped, not counted as a passed product check.

### Decision

Hold publication. Automatic local and hosted checks passed, but the actual
125%/150% and mixed-monitor visual matrix remains pending. Real native
save-consent/restart confirmation was not repeated with this candidate;
earlier maintainer confirmations remain historical, not new alpha.54 evidence.
The 7-second-class same-profile relaunch latency remains an explicit known
limitation, and passing cancellation samples do not erase historical failures.

No release tag, public publication or beta promotion was performed. Public
alpha.51 remains the recorded public preview. This follow-up changes only
readiness documentation: API, DB, runtime, storage and repository hygiene
rules are unchanged. No additional runtime patch or version bump was made.
