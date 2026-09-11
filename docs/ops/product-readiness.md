# Product Readiness

- Status: Alpha active; beta not approved
- Decision: ADR 0019
- Owner: Project maintainer

## Required Beta Checks

| Check | Required evidence | Current boundary |
| --- | --- | --- |
| Public consumer path | Exact release URL and ZIP digest; source-free init, validate, doctor, build twice, inspect, launch | Refresh for the next public candidate |
| File Notes behavior | Real open, edit, save, save-as cancellation, permission denial, close/reopen and draft recovery using disposable files | alpha.51 fixes restored-handle permission denial; a copied-profile Save as completed locally, but complete real picker and persistence checks remain unverified |
| Development loop | Source run, edit/reload, default debug-off, preserved profile and app ID | CLI debug forwarding and native debug startup pass; interactive reload remains unverified |
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
Save-to-existing, cancellation, denied consent and subsequent restart recovery
still need full manual coverage. The public alpha.49 release is unchanged.

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
