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

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
