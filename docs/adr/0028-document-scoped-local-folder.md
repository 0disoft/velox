# ADR 0028: Document-scoped local folder selection

- Status: Accepted
- Date: 2026-10-03
- Owner: Project maintainer

## Decision

Add independent opt-in `folder.read`, with defaults unchanged (no permission).
`folder.select` accepts only `{}` and opens one host-owned folder dialog outside
the WebView callback. It uses the existing IFileOpenDialog adapter with
[folder selection options](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/ne-shobjidl_core-_fileopendialogoptions).
Cancellation returns `cancelled: true`, without replacing the previous target.
Successful selection returns only a folder basename and positive uint32 target.
No caller-supplied path is accepted or filesystem path/identity returned.

Validate local fixed/removable/RAM drives, clean absolute drive paths, every
existing path component, directory attributes and final handle path. Reject
network/device/stream paths, links/junctions/reparse points, offline and encrypted
directories, and ordinary files. Windows short-name spellings are accepted only
when their expanded long path equals the handle's final path. Folder root labels
use `Volume root`, not an exposed drive path. Normal Windows access checks apply.

Retain one host-private path plus volume/file identity, bound to the current
document generation. Tokens are not reused within a process. A new successful
selection replaces the previous token; failure/cancellation preserves it.
`folder.release` accepts only `{target}` and revokes that exact connection.
Navigation, reload and shutdown clear it. Queued operations are generation and
revocation checked. No grant is persisted; restart requires new selection.
No file handle stays open between operations.

## Bounded Listing Increment

Selection alone does not return entries, authorize file contents, child-folder
navigation, filesystem writes, recursive traversal or directory watching.
Beta.17 adds `folder.list({target})` under this permission, deferred outside
WebView callbacks. Reopen/validate the stored directory path and verify its
volume/file identity. Enumerate from that checked handle with the existing
Go Windows `File.ReadDir` implementation; do not resolve or open child paths.
Materialize at most 129 immediate entries, using the extra one only to detect
truncation; consider the first 128. Return names and file/directory kinds only.
Exclude reparse, offline and encrypted entries without following them, and
report examined exclusions as `skipped`. Ordinary hard-linked names can appear
but confer no content or write capability.

Cap the JSON result at 32 KiB including escaped names. `truncated` reports the
entry/byte bound; there is no pagination, recursive scan, total-count, sorted
order or stable-snapshot claim. Changes can occur during enumeration. Deleted
or replaced directories return `FOLDER_TARGET_INVALID`; queued release,
navigation and shutdown invalidate operations before and after reading.
No path or child-entry identifier becomes an implicit file-read/write grant.

## Cost and Evidence

Reuse the existing UI dispatch and dialog COM adapter; no listener, timer,
watcher, worker or dependency is added. An opted-out host allocates no folder
controller. Selection and identity checks are on-demand local filesystem work,
not a latency guarantee for slow devices or filesystem drivers. Measure the
built executable rather than claiming zero size increase.

Tests cover default denial, unknown/path parameters, cancellation, replacement,
release, generation mismatch, queued revocation, shutdown, token exhaustion,
actual dialog configuration, local/short-path identity, and unsupported paths.
Listing tests cover empty/Unicode folders, immediate entries, entry/escaped-byte
limits, child reparse/offline exclusion, identity replacement/deletion, queued
revocation and no retry. Folder Browser is a dependency-free reference consumer.
Actual native user selection is separate manual evidence. This does not change
the release-channel admission decision.

## Rollback

Remove `folder.read` from a consuming manifest to disable the capability.
Removal of the methods/controller leaves the existing file APIs unchanged.
