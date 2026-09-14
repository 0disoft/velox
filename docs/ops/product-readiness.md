# Product Readiness

- Status: Alpha active; beta not approved
- Decision: ADR 0019
- Owner: Project maintainer

## Required Beta Checks

| Check | Required evidence | Current boundary |
| --- | --- | --- |
| Public consumer path | Exact release URL and ZIP digest; source-free init, validate, doctor, build twice, inspect, launch | alpha.51 public verification run 34585168947 passed; exact source and digest are recorded in release.md |
| File Notes behavior | Real open, edit, save, save-as cancellation, permission denial, close/reopen and draft recovery using disposable files | Native cancellation/retry, denied-write protection, post-denial draft restoration and manual Save as with exact disk readback passed; same-file retry interaction was not separately observed |
| Development loop | Source run, edit/reload, default debug-off, preserved profile and app ID | Private-profile CDP normal reload updated HTML, CSS and JS with stable origin; an earlier smoke failure remains unexplained |
| Windows lifecycle | Bounded shutdown, immediate relaunch, initialization cancellation, no residual process/profile lock; bind runtime and source/artifact versions | alpha.40 public lifecycle and hosted source-fork cancellation are historical evidence, not proof for a new ZIP |
| Security and data integrity | No unresolved critical issue; permission, origin, overwrite and recovery checks pass | Preserve existing security gates and unsigned-alpha warnings |

A visible window or readiness callback does not prove working file pickers,
correct rendered content, durable edits, or reload behavior. Use disposable
files and a private profile for validation; never overwrite a user's files.
Record failures and denied operations, not only successful paths. A failed or
unverified required check keeps beta held. Do not replace a real interaction
with a mock and label the gate complete.

## Delivery Record: 2026-09-10

- File Notes now records only the submitted save snapshot as the on-disk
  baseline. Later edits stay dirty. File operations are serialized, and draft
  restore locks editing until completion.
- Eleven File Notes model/application tests pass using deterministic API
  doubles, including cancellation, denial, write failure and restored drafts.
- Native File Notes smoke passed normal and debug source startup, portable
  startup, inspection and byte-identical repeated builds locally.
- `velox run --debug` exposes existing host tools without changing app ID,
  profile selection, native permissions or packaged defaults.
- Alpha.49 publication and verification evidence is recorded in `docs/ops/release.md`.
- Final local Windows startup smoke failed once, then passed unchanged; its
  intermittent failure remains unexplained and is retained in the release record.
- Real native picker actions, interactive reload and durable restart recovery
  are not claimed by the automated tests above.

### File Save Permission Repair

Manual testing on 2026-09-10 reported `Write permission was not granted` for
Save and a platform-denied `showSaveFilePicker` call for Save as. The host's
global permission denial also denied WebView2 FileReadWrite (kind 8).
The alpha.50 gesture gate was incomplete: on 2026-09-11 a copied existing
profile emitted FileReadWrite requests with `IsUserInitiated=FALSE` while
restoring serialized handles. The host denied them before a picker action.
Clearing the saved write-guard entry alone did not fix this behavior.

The alpha.51 candidate delegates trusted-origin requests to browser DEFAULT
without treating that flag as an authorization decision. Browser activation
checks and consent remain in force; the host never returns ALLOW for these
requests. COM regressions for restored handles failed before the change and
pass after it. Foreign/missing origins, failed kind/origin reads, opt-out and
unrelated permissions remain covered.

With a copy of the affected profile, native diagnostics changed from kind 8 /
state 2 (DENY) to trusted-origin kind 8 / state 0 (DEFAULT). The user completed
Save as in the diagnostic app; the UI reported Saved to file and the resulting
file existed on disk. The original profile Preferences digest was unchanged.
Diagnostic instrumentation was an uncommitted Go overlay, not release code.
The immutable public alpha.49 release was not modified by this repair.

### Maintainer Confirmation: 2026-09-11

The maintainer confirmed that saving worked in the normal alpha.51 app, then
confirmed close/reopen recovery. This records user-reported manual evidence
for Save, Save as and restart recovery, separately from the copied-profile
diagnostic and automated tests. Those successful paths do not need another
repeat solely to update this record.

The existing application tests additionally verify that canceling a save
picker retains the draft and permits a later save, and that denied permission
opens no writable stream, preserves edits and permits retry when the browser
later grants access. These are browser API doubles, not native permission UI
evidence. The application does not override a browser denial or reset profile
permissions. Native cancellation and denied-consent recovery are still pending;
complete real picker and persistence checks remain unverified until those
remaining paths are covered. Use a disposable profile for a native denial test.

That confirmation update changed evidence and tests only. The runtime stayed
at alpha.51; publication followed separately as recorded below.

### Alpha.51 Delivery and Source Reload: 2026-09-11

The unsigned alpha.51 release pins source
`f18d7f3958c136b1f673b93255916771db3cde15`. Tag evidence run 34584656621,
publication run 34584828937 and public download verification run 34585168947
passed. The published ZIP digest and independent provenance binding are
recorded in `docs/ops/release.md`. This is same-repository consumer evidence,
not an external user attempt or beta approval.

A local `velox run --debug` smoke copied File Notes into a disposable project
and used a private profile. After modifying HTML, CSS and JavaScript, a normal
CDP Page.reload updated all three rendered probes while preserving origin.
Hard reload was not needed in that successful run. The smoke did not modify
the maintainer's open application, original project or profile.

An earlier run failed the combined reload condition before recording its
rendered state. The diagnostic was improved to retain normal and hard reload
results, and the next run passed normal reload. The first failure remains
unexplained; this is one successful browser-driven reload, not proof of
reliable repeated reload or a manually exercised keyboard shortcut. No
runtime reload fix was made. Native save cancellation, denied-consent recovery
and current-artifact lifecycle checks still keep beta held.

### Native Save Cancellation: 2026-09-11

An isolated session used the checksum-verified public alpha.51 ZIP, the
repository File Notes example and a fresh private profile. A real Save as
dialog was canceled with Escape. The editor retained the complete probe,
showed `Unsaved changes` and `Save canceled.`, and re-enabled its commands.
The subsequent Save opened another native picker and wrote `retry.md` in the
disposable test directory. The UI showed `Saved to file`; a separate disk
read confirmed the exact probe content.

Opening and editing the disposable `input.md` then clicking Save displayed
the browser's native write-consent prompt. The original file remained
unchanged while awaiting consent. Permission prompt choices require a human;
the agent did not grant, deny or reset permissions. At that point, denial and
later recovery were unverified; subsequent observations follow below.

The initial session hit its six-minute deadline while awaiting the manual
permission choice and was terminated by the wrapper. This is not a graceful
shutdown pass. Relaunching the same private profile restored the unsaved
63-character draft and its filename. Save then displayed the browser's
restored-file permission prompt; no permission choice is inferred from that
successful draft restoration.

The maintainer subsequently allowed the restored-file prompt, and `input.md`
saved successfully. A separate disposable README copy was opened and edited
to exercise a new write-consent request. The maintainer canceled that request;
the UI reported `Save failed: Write permission was not granted.` while
retaining the 783-character dirty draft. The on-disk copy's SHA-256 still
matched the source example README. This is observed native denied-write
protection, not an API double.

The second bounded session also reached its deadline. A third launch restored
the denied-write draft and its probe unchanged. Automated Save and Save as
clicks did not visibly advance the UI in that session; successful saving after
denial remains unverified. Automated title-bar and keyboard close attempts
also produced no visible change, while Windows reported the host responding;
input delivery versus application behavior was not isolated. The third session
also reached its deadline. These timeouts are harness cleanup events, not
evidence of a runtime crash or graceful lifecycle success.

### Input Delivery Investigation: 2026-09-11

A ninety-second passive DOM probe ran against a copy of the denied-write
test profile with the same public alpha.51 host. All eighteen CDP samples
returned successfully, without evaluation exceptions; the document command
buttons remained enabled. The native automation tool was asked to click
Save as, but the captured trusted pointer/mouse/click sequence targeted
`HEADER`, not `save-as-document`. No save-button click was recorded. Focus
also changed during observation, and a later screenshot was occluded by
another foreground application; no input was sent to that application.

This isolates the observed attempt to input targeting before the save
handler, not a reproduced save-handler failure. It does not establish the
underlying coordinate/focus cause or prove that every earlier attempt failed
for the same reason. Keyboard and manual-click comparison remain unverified.
No runtime fix or permission reset was applied. The temporary probe was
removed after collecting evidence and its process tree was stopped. A real
post-denial save remains required before completing that readiness check.

### Manual Post-Denial Save: 2026-09-11

The maintainer reopened the existing isolated alpha.51 session and confirmed
saving after being asked to use Save as. The new disposable file
`recovered-after-denial.md` contained exactly the restored 783-character probe
including `NATIVE-DENIAL-PROBE`, verified against the source example text with
the observed insertion and normalized editor line endings. Its 783 bytes had
SHA-256 `a165087afffcb95c18a73e2bb3b8330dac9e8b4e2c01f69fff739a4d524b850b`.
The session exited with code 0 before its deadline; it was not killed by the
wrapper. No automated clicks were used in this session.

The disposable original README also contained the same edited text after
this manual session. Its exact save interaction was not separately observed,
so this record claims the verified new-file recovery path, not a particular
same-file permission regrant sequence. The prior unchanged-original check
applies to the earlier denial instant, not the later manual save session.
The tracked example was unchanged. This closes the pending post-denial Save
as check without claiming the input automation coordinate issue is fixed.
Current-artifact lifecycle stress and repeated reload evidence remain separate
beta requirements; no runtime change or new release is included.

### Public Host DPI Diagnosis: 2026-09-12

A fresh extraction of the checksum-verified public alpha.51 ZIP was launched
with a disposable File Notes project and profile. Read-only Windows queries
returned process awareness 0 and window awareness 0 (DPI unaware), window DPI
96, and monitor scale 125 percent. All queried HRESULTs succeeded. The host
SHA-256 was `2b1da59f1ba61f5aac554a9ee3dc272a6edf42991f628015ddbb56bd275cd0b9`.
The probe stopped only its own process tree after collecting the result; no
display, compatibility or permission setting was changed.

This confirms that the public host is DPI unaware on the tested scaled
display. Windows can bitmap-scale such windows, making this a supported
explanation for the reported blurred text, rather than evidence that the
16px editor font itself is too small. See Microsoft's
[high-DPI guidance](https://learn.microsoft.com/en-us/windows/win32/hidpi/high-dpi-desktop-application-development-on-windows).
Per-monitor awareness, window resizing on DPI changes and WebView bounds need
a separate implementation and verification unit. No font or runtime change
was made in this diagnostic unit. Visual before/after comparison, mixed-DPI
monitor transitions and any relationship to the earlier automation coordinate
mismatch remain unverified; no Tauri/Wails parity claim is made.

### Per-Monitor DPI Candidate: 2026-09-12

The local alpha.52 candidate configures per-monitor V2 awareness before host
window creation, falling back to V1 only when the V2 context is unsupported.
An incompatible preconfigured awareness mode fails explicitly. Logical
96-DPI window dimensions and size limits are scaled; WM_DPICHANGED applies
the suggested physical rectangle and refreshes WebView bounds and position.
The editor font remains unchanged.

On the same 125-percent display, the local candidate reported process and
window awareness 2 (per-monitor) and window DPI 120, replacing the public
alpha.51 unaware/96 result. The candidate host SHA-256 was
`7529e3d80280dbecda8153a7a2a3aa18896010e9275a6ff0ecc32028e522d227`.
Focused host/CLI/build/inspect/runner/hygiene tests and the fork window/COM
suite passed, including 100/125/150/200-percent size arithmetic and a native
window test of suggested DPI-change bounds with a fake browser adapter.

The native window test sends a synthetic DPI-change message; it does not
prove actual cross-monitor WebView rasterization or visual sharpness. Real
mixed-DPI monitor movement, Server 2016 fallback execution and a visual
before/after comparison remain unverified. The candidate is local only;
public alpha.51 is unchanged, no release was published, and beta stays held.

### Alpha.52 Lifecycle Recheck: 2026-09-12

Local checks at source `3295d573017e978607472c86c0d2988a87b8b650`, using
Windows amd64, Go 1.26.4 and WebView2 152.0.4191.66, did not pass the
lifecycle gate:

- `velox_native_cancellation_test`: four of six source-fork cases passed.
  Controller-pending repetitions 2 and 3 failed with `late controller browser
  did not exit` at the ten-second process-exit boundary. Callback references
  had drained and destroyed-state assertions had passed; those cases never
  reached the profile-release assertion. The environment-completion cases
  and controller-pending repetition 1 passed. This test does not run the
  production host's DPI initialization, so it does not establish a DPI
  regression. A subsequent process inventory found no matching test browser
  still running; that does not turn the deadline failures into passes.
- `velox_build` passed. The rebuilt alpha.52 host SHA-256 was
  `cc44af54ecf500415e2f7ee7136664b58d802c3eaf29ac133a2937870d7ed24c`.
  This differs from the earlier local ZIP host above; the following result
  binds to this source rebuild, not the earlier ZIP or a public download.
- `velox_design_lifecycle_test`: nine of ten fresh/immediate same-profile
  pairs passed. Sample 0 failed at `immediate-launch` with `HOST_RUN_FAILED`;
  its first host reached ready and exited, but no immediate-launch result was
  retained. The harness drops the underlying run error at this boundary,
  leaving the specific failure cause unresolved. Successful samples observed
  both browser exits and profile removal. Raw v3 evidence covers
  07:49:38-07:52:07 UTC; no hosted-run claim follows from this local check.

Preserve these failures rather than increasing deadlines or replacing them
with unchanged passing retries. The diagnostic follow-up now logs each failed
sample's phase, stable error code and underlying cause in the Go test log;
the v3 JSON shape remains unchanged. Known workspace, temporary and user-profile
roots are masked, messages are quoted and limited to 4,096 bytes. Host failure
messages include the observed exit code (or `unavailable`), and distinguish a
requested test cleanup kill from a natural nonzero exit. Native cancellation
logs include browser PID, exit observation, elapsed wait, poll count, final
Windows wait status/error and callback references before asserting failure.
Existing timeouts and pass conditions remain unchanged. Focused diagnostics
tests cover a failed launch, exit code 6, bounded/masked messages and browser
wait timeout/error/success classification; this does not resolve or supersede
the observed native failures. Next isolate browser-exit delay and the detailed
immediate-launch error before deciding whether runtime, fixture or environment
changes are needed. Publication of alpha.52 remains held;
repeated reload, visual comparison and mixed-monitor checks were not run in
this unit. Public alpha.51 and the beta hold are unchanged.

### Native Cancellation Diagnostic Rerun: 2026-09-12

One six-case run at source `4213851c087e5e81852dd3b810e3ba9357a07934`
passed with the same WebView2 152.0.4191.66, Go 1.26.4 and Windows amd64.
The three controller-pending browser waits observed exit after 1,381, 193 and
194 ms; each ended with wait status `0x00000000`, no wait error and zero
callback references. All six cases passed profile cleanup. No matching test
browser remained in the post-run process inventory. Total native test time
was 3.10 seconds, with the existing ten-second exit deadline unchanged.

The earlier two deadline failures were not reproduced, not fixed or erased.
This single source-fork rerun does not identify their cause or establish
production-host or release-ZIP reliability. No further unchanged retries were
run. The separate immediate-relaunch failure is still unresolved; its new
diagnostics are the next bounded investigation. Publication and beta remain
held. No runtime or version change follows from this observation.

### Same-Profile Diagnostic Smoke: 2026-09-12

One `velox_startup_smoke` run at diagnostic source
`577dc1324e013b0999dca2c691d17ddca7076fd1` passed using the unchanged local
alpha.52 rebuilt host SHA-256
`cc44af54ecf500415e2f7ee7136664b58d802c3eaf29ac133a2937870d7ed24c`.
First startup reached ready in 772.8 ms and host exit in 115.4 ms. Immediate
same-profile startup reached ready in 1,659.5 ms and host exit in 94.0 ms;
browser exit and profile removal were observed after about 6.65 seconds.
The included security-policy and missing-runtime checks also passed. The
complete smoke took 17.73 seconds; no deadlines or pass criteria changed.

The earlier immediate-launch failure was not reproduced, and this one-pair
pass does not identify or fix its cause. It also does not validate the earlier
release ZIP, File Notes interaction or development reload. Those artifact and
workflow checks remain separate. No additional unchanged reruns or runtime
changes were made; alpha.52 publication and beta remain held.

### Pinned Alpha.52 ZIP Lifecycle: 2026-09-12

Three fresh/immediate same-profile pairs passed using the host extracted into
a new isolated cache directory from the unchanged local candidate ZIP:
3,605,419 bytes, SHA-256
`b2cb6b44152a4da5c38ea4925deb53ce4ec4b016418228062b69a88bfce0957c`.
The extracted host hash was
`7529e3d80280dbecda8153a7a2a3aa18896010e9275a6ff0ecc32028e522d227`.
Release identity and every manifest-listed artifact's size and digest were
checked before execution; ZIP and host digests were unchanged afterward.
No host rebuild or public download was used.

Harness source `0fc60070f0f3b63811cd9c98c64c5ea67e8ede0d` used the source
hello fixture and WebView2 152.0.4191.66 on local Windows amd64. Raw v3
evidence covers 14:18:03-14:18:47 UTC, SHA-256
`b2311d4b8b3c804523b7fad01a8345939f8652d409c1efde3110b7a004b4c39c`.
All six hosts reached ready and exited; both browser exits and profile
removal were observed for every pair. Fresh startup took 574-638 ms,
immediate startup 7.07-7.21 seconds, host exit 67-170 ms, and profile release
6.42-7.09 seconds. Total native test time was 43.93 seconds.

The roughly seven-second immediate-startup delay remains a usability concern;
passing the bounded lifecycle check does not establish performance parity or
identify the cause of that delay. Earlier intermittent failures remain in the
record. This is local extracted-host evidence, not File Notes interaction,
CLI launch, development reload, hosted stress or public-download evidence.
No additional retries, runtime edits or publication were performed. Reload,
visual checks and the release decision remain pending; beta stays held.

### Alpha.52 Normal Reload Failure: 2026-09-13

A private File Notes copy and fresh profile were launched through the local
alpha.52 `velox run --debug` CLI. The verified CLI SHA-256 was
`3b48f3d37d2baf2f476124514c480421266a59ee6f35f25fb7f418a82e1471bf`;
the host was the same `7529e3d8...522d227` artifact recorded above.
The probe added separate HTML, CSS and JavaScript markers only to the copy.

The first ordinary CDP `Page.reload` with `ignoreCache: false` failed:
HTML changed from `before` to `after-one`, while JavaScript stayed `before`
and computed CSS stayed `rgb(200, 10, 20)` instead of `rgb(10, 120, 30)`.
The virtual origin remained unchanged. The planned second change was not
attempted after this failure, and no hard reload or cache-disabling override
was used to turn it into a pass. The owned process tree was stopped
successfully. The result covers 06:12:19-06:12:26 UTC and is retained under
the ignored `reload52-1789279939175` evidence directory.

Source inspection shows `internal/webview2/runtime_windows.go` using virtual
host folder mapping without a separate debug cache policy. Cache behavior is
a candidate explanation, not yet a proven root cause; conditional requests,
file timestamps and development-mode cache handling need a targeted comparison.
The next implementation decision must preserve production asset serving and
origin/security boundaries. No runtime edit was made in this diagnostic unit.
Normal development reload is now a confirmed failing workflow for this run;
earlier successful checks do not supersede it. Alpha.52 publication and beta
remain held, and visual confirmation remains pending.

### Reload Cache Comparison: 2026-09-13

One bounded comparison used the same hash-verified alpha.52 CLI and host,
a copied File Notes project and a fresh private profile. Two separate CSS/JS
pairs changed from equal-length `base0` content to `next1` content: pair A
kept its original filesystem modification time, while pair B advanced by
exactly 2,000 ms. No system clock, source example or production setting changed.

After ordinary reload, HTML was current but both pairs still showed old JS
and CSS. All four resource requests emitted CDP `requestServedFromCache`,
reported status 200, `fromDiskCache: false` and `fromServiceWorker: false`,
and retained the original Last-Modified value. Captured cache-related request
headers contained no validators. A following `ignoreCache: true` reload of
the same unchanged files updated both pairs; no response was marked as a
disk/service-worker cache response, and pair B returned the advanced
Last-Modified value. No Cache-Control or Expires header was present in the
captured response headers. The virtual origin stayed unchanged.

This comparison localizes the stale-resource behavior to browser cache reuse
on normal reload, rather than a failure to write or read the edited files.
Advancing timestamps alone did not prevent it, so same-second modification
time precision is not a sufficient explanation. The next fix should control
cache reuse only in explicit development mode while preserving production
virtual-host serving and origin/security boundaries. The cache-bypassed
success is diagnostic evidence, not a normal-reload pass or a product fix.

Evidence covers 14:53:56-14:53:58 UTC under the ignored
`reload-cache-compare-1789311236574` directory; result SHA-256 is
`14cb4c764b3eabd7022584eb5f64d19f44041e073ac4de964f4190dde85e8b4d`.
The owned process tree was stopped successfully and the binaries were
unchanged. No runtime modification or publication occurred. Alpha.52 and
beta remain held pending the development-mode fix and remaining visual checks.

### Development Cache Fix: 2026-09-14

Local alpha.53 requests `Network.enable` followed by
`Network.setCacheDisabled` with `cacheDisabled: true` through WebView2's
in-process protocol API when constructing an explicit development WebView.
Production mode makes neither call. Virtual-host folder mapping, permission
and navigation rules, IPC and persistent browser storage are unchanged; this
policy opens no debugging listener. Dispatch HRESULT failures abort window
initialization. The commands are asynchronous with no retained completion
callback; the native regression checks their observed effect, not just dispatch.

The first candidate, which sent only the cache policy without enabling the
Network domain, still failed the first ordinary reload. That failed result is
retained under `normal-reload-1789312659928`; it is not counted as a pass.
After enabling the domain, `scripts/dev-reload-smoke.ts` passed two consecutive
HTML/CSS/JS edits using only `Page.reload` with `ignoreCache: false`. The script
does not send a cache-disabling command or perform a hard reload. It verifies
release manifest hashes, uses a private example/profile, checks the same
origin and retains the result before stopping its owned process tree.

The successful result covers 2026-09-13 15:20:14-15:20:16 UTC under
`normal-reload-1789312814831` (2026-09-14 in Korea). The tested host SHA-256 is
`0ee10fc617554fcf17b1f1c26a047bb7167b479c20e161cdc178e76408de0490`.
Production startup, same-profile relaunch, missing-runtime handling and
security-policy smoke also passed. Immediate relaunch still took 6.92 seconds;
this patch does not claim to fix that delay or historical cancellation failures.

Focused fork/COM, runtime, version-fixture and hygiene checks cover production
no-op behavior, invalid lifecycle state, exact protocol arguments, and native
dispatch failure. Version fixtures now use alpha.53; the public alpha.51 release
is unchanged. No publication or beta promotion occurred. Mixed-DPI visual
confirmation and the release decision remain pending.

### Relaunch Profile Comparison: 2026-09-14

Reanalysis of the three retained alpha.52 ZIP lifecycle samples places the
largest relaunch interval between `environment-created` and
`controller-created`: 6,684.62-6,917.44 ms. The latter marker follows core
WebView acquisition and event/security handler setup, so this interval is not
an isolated measurement of the asynchronous controller creation call.
Navigation dispatch to DOM plus two animation frames took 165.28-294.21 ms.
The harness started the second host 7.53-25.54 ms after the first host exited;
it did not insert a seven-second prelaunch wait. The first browser exited
6,403.48-6,460.77 ms after the second process started, and readiness followed
669.68-745.58 ms later. These timings are alpha.52 observations, not alpha.53
phase measurements. The original evidence SHA-256 remains
`b2311d4b8b3c804523b7fad01a8345939f8652d409c1efde3110b7a004b4c39c`.

One bounded alpha.53 comparison then used the unchanged host SHA-256 above,
the `examples/hello` fixture, harness source
`99fa5e3cd2ad804242a965701c5fec0389681f6e`, and WebView2 `152.0.4191.66`
on local Windows amd64. `TestStartupProfileComparisonEvidence` ran one
same-profile trial followed by one fresh-profile trial, four host launches
in total, with isolated test profiles and no runtime edits.

| Observed boundary | Same profile | Fresh profile |
| --- | ---: | ---: |
| Second host start to ready | 6,903.07 ms | 547.15 ms |
| First browser exit after second host start | 6,380.15 ms | 6,430.17 ms |
| Second ready after first browser exit | 522.92 ms | -5,883.02 ms |

The fresh-profile host became ready while the previous browser was still
alive. The observed 6,355.92 ms difference supports a same-profile reuse
dependency during browser shutdown; it does not establish why WebView2
retains its browser process for about 6.4 seconds. This is one sequential
comparison, not an order-balanced benchmark or a population percentile,
despite the harness summary's P50 field names. Warm-cache and trial-order
effects remain possible. No user profile was rotated, no browser was
force-terminated as a remedy, and persistent storage behavior is unchanged.

The test passed in 24.05 seconds. Evidence covers 06:10:50-06:11:14 UTC under
the ignored `profile53-compare-e73b203ded7e4b519e0a10d8e06352e2` directory;
result SHA-256 is
`b028fba33e0e857d75e561b5e55d6f04e68d02f7d1609321f50f56df2345a5c1`.
The host hash matched before and after execution. Next diagnosis should
separate controller callback entry from setup completion and inspect the
native close/release lifecycle before selecting a storage-preserving fix.
Historical cancellation failures, mixed-DPI visual confirmation and the
release decision remain open. This record does not publish alpha.53 or
promote beta; API, DB, runtime and repository hygiene rules are unchanged.

### Native Callback Phase Isolation: 2026-09-14

The opt-in `TestNativeRelaunchPhases` diagnostic wraps the fork's existing
controller-completion implementation in test code. It timestamps callback
entry before forwarding to the unchanged implementation and observes the
existing `controller-created` marker after setup. Each initialization runs in
a separate, timeout-bounded test subprocess, using an isolated profile and
hidden native window. Browser-exit waits occur during cleanup after both
launches, not as a prerequisite for relaunch. Ordinary fork tests skip this
native diagnostic. No product callback, lifecycle order or timeline schema
changed, and no release version bump is needed for this test-only addition.

One same-profile pair followed by one fresh-profile pair passed on local
Windows amd64 with WebView2 `152.0.4191.66` and Go `go1.26.4`:

| Initialization interval | Same first | Same second | Fresh first | Fresh second |
| --- | ---: | ---: | ---: | ---: |
| Environment marker to controller callback | 286.639 ms | 234.121 ms | 330.323 ms | 369.289 ms |
| Callback entry to setup marker | 0.000 ms | 0.000 ms | 0.590 ms | 0.000 ms |
| Destroy call duration | 12.972 ms | 9.877 ms | 19.311 ms | 14.496 ms |

Zero-valued intervals mean no difference resolved by this measurement, not
zero execution cost. All four callbacks reported success exactly once;
`Destroy` cleared the three owned COM interface fields and the callback
reference registry count was zero immediately afterward. Recorded shutdown
order was event removal, controller close, WebView release, controller
release, then environment release. Explicit close is consistent with
[Microsoft's controller lifetime guidance](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2controller#close),
but this observation does not verify every native COM reference or the close
HRESULT: the current close wrapper does not inspect that HRESULT and Destroy
discards its returned error. There is no evidence here that close failed.

The roughly seven-second release-host delay did not reproduce in this
initialization-only experiment. This test does not navigate, configure the
full product security policy, wait for DOM readiness, or use the release
executable. It also explicitly initializes and uninitializes its test STA,
unlike the product path's package-initialization lifetime. These differences
prevent transferring the fast setup timings or zero callback references to
the failing full-app path. The earlier alpha.53 same-profile result remains
unresolved; neither a generic COM leak nor an unavoidable WebView2 delay is
established. The next useful experiment must retain the production navigation,
settings and shutdown path while observing the same callback boundaries,
rather than repeat the minimal test or rotate user profiles.

The native comparison passed in 5.82 seconds and fork unit regressions passed.
Evidence is retained under the ignored
`native-phase-2c82cc79b3cc4c5891beebddd395c91c` directory, bound to base commit
`ab69cc70f1f639402d5f563eaa8e973954583f7c` and diagnostic source SHA-256
`cf54a8633b2bdf202e76c2f9a6cf35e15b9440d1df2e5cfd83c5f652967e6fe0`.
The native log SHA-256 is
`3fb22e75047cf00b5315a372c5b8412206107bac2760a4fe03fbfb4cbca4480a`.
No production binary was rebuilt or published. API, DB, runner selection,
persistent data and repository hygiene rules are unchanged. Full product
regression, historical cancellation failures and visual DPI checks were not
rerun in this bounded diagnostic unit.

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
