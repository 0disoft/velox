# Product Readiness

- Status: Alpha active; beta not approved
- Decision: ADR 0019
- Owner: Project maintainer

## Required Beta Checks

| Check | Required evidence | Current boundary |
| --- | --- | --- |
| Public consumer path | Exact release URL and ZIP digest; source-free init, validate, doctor, build twice, inspect, launch | alpha.51 public verification run 34585168947 passed; exact source and digest are recorded in release.md |
| File Notes behavior | Real open, edit, save, save-as cancellation, permission denial, close/reopen and draft recovery using disposable files | Maintainer confirmed Save, Save as and restart recovery on alpha.51; native cancellation and denied-consent recovery remain separate checks |
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

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
