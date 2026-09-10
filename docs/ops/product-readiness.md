# Product Readiness

- Status: Alpha active; beta not approved
- Decision: ADR 0019
- Owner: Project maintainer

## Required Beta Checks

| Check | Required evidence | Current boundary |
| --- | --- | --- |
| Public consumer path | Exact release URL and ZIP digest; source-free init, validate, doctor, build twice, inspect, launch | Refresh for the next public candidate |
| File Notes behavior | Real open, edit, save, save-as cancellation, permission denial, close/reopen and draft recovery using disposable files | Manual testing found Save and Save as blocked by host permission denial; alpha.50 corrects the policy, but real picker and persistence checks remain unverified |
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
The alpha.50 source candidate delegates only user-initiated requests from the
trusted app origin to browser consent. It does not automatically grant access,
change the native IPC method table, or clear persisted profiles.
COM callback tests cover default consent, foreign/missing origins, missing or
invalid gestures, failed metadata reads, opt-out, and unrelated permissions.
Actual picker, existing-profile permission recovery, disk writes, cancellation,
and close/reopen recovery still require manual verification. Do not overwrite
or close the user's running editor to install this candidate while edits remain
unsaved. The public alpha.49 release is unchanged.

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
