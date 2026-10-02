# ADR 0027: Document-scoped save target

- Status: Accepted
- Date: 2026-10-03
- Owner: Project maintainer
- Amends: ADR 0026 (explicit session-only target reuse)

## Decision

Keep `saveText` and `file.commitSave` as per-call native selection, without a
reusable grant. Add opt-in connected behavior under the existing `file.save`
permission: `saveTextAs` stages UTF-8 text and invokes `file.commitSaveAs`.
After native selection, overwrite confirmation and successful writing, verify
the saved bytes and return a positive uint32 `target` alongside name/bytes.
`saveTextTo(text, target)` commits another bounded upload to that target without
showing a picker. No request accepts a path. `file.releaseSaveTarget` revokes it.

Retain only one target per host, bound to the current document generation.
Its token is never reused within the process. It identifies a host-private path
and baseline, not a persisted OS permission. A successful connected Save as
replaces the previous target; cancellation preserves it. Accepted navigation,
reload and shutdown drop the target and staged text. Apps must also release it
when replacing the editor document without navigating, for example on New.
No target, path or baseline is written to profile storage or returned by Open.
Restored drafts require a new Save as. There is no reopen-and-save grant.

## Conflict and Data Integrity Boundary

Before reusing a target, validate the existing local file under ADR 0026's
path, link, drive, attribute and sharing rules. Read at most 2 MiB from its
verified handle. Compare volume/file ID, byte size, last-write time and SHA-256
with the last successful save's baseline. Deletion, replacement or modified
contents return `FILE_CHANGED` before creating a save temporary file. Hashing
also detects same-size edits with a restored timestamp. Do not recreate a
deleted target or offer silent force-overwrite. Invalid/released/stale targets
return `SAVE_TARGET_INVALID`. Keep the app's editor contents on failure.

Use the existing flushed sibling-file replacement and recovery behavior.
After writing, verify the saved body matches the submitted bytes before
creating or refreshing the connection. If that verification fails, the write
may already have occurred; return an error without granting a new baseline.
Keep the prior baseline after a failed Save so another reuse cannot silently
overwrite a conflicting file. Save as remains an explicit user-approved path
to a different target or overwrite, with its native prompt.

Write handles deny shared writes, and target identity is rechecked immediately
before replacement. Windows replacement is not an atomic compare-and-swap:
same-user path mutation between checks and replacement remains outside the
protection claim. This does not merge concurrent edits, detect every historical
edit later reverted to identical bytes/metadata, or guarantee power-loss recovery.
ACL changes can also make a saved target unusable under normal OS permissions.

## Cost and Validation

Reuse the 64 KiB message bound and 2 MiB text staging. Baselines contain only
identity/size/time, a 32-byte digest and one private path; no file handle stays
open between operations. Hash and metadata reads occur on demand at saves,
on the existing UI thread. No watcher, timer, worker, new dependency or idle
polling is added. Larger saves incur extra bounded reads; no latency benefit
is claimed. Measure built-host size, not just source size.

Cover repeated save and baseline refresh, same-size/time edits, file replacement,
deletion, cancellation retaining the previous target, explicit release of queued
writes, stale document generations, changed post-write readback, permission and
parameter denial, helper routing and navigation/shutdown cleanup. The Text Writer
example adds New, Save and Save as, keeps buffers on errors and revokes on New.
File Notes migration and real dialog/conflict interaction are separate checks.

## Rollback

Remove the connected methods/helper usage and retain ADR 0026's per-call save
behavior. Default permissions remain empty; removing `file.save` disables all
text-write methods. Existing read-only `file.openText` remains unchanged.
