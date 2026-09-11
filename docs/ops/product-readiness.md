# Product Readiness

- Status: Alpha active; beta not approved
- Decision: ADR 0019
- Owner: Project maintainer

## Required Beta Checks

| Check | Required evidence | Current boundary |
| --- | --- | --- |
| Public consumer path | Exact release URL and ZIP digest; source-free init, validate, doctor, build twice, inspect, launch | alpha.51 public verification run 34585168947 passed; exact source and digest are recorded in release.md |
| File Notes behavior | Real open, edit, save, save-as cancellation, permission denial, close/reopen and draft recovery using disposable files | Native cancellation/retry, denied-write protection and post-denial draft restoration passed; post-denial successful saving remains pending |
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

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
