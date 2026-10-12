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

## Immediate Text Read Increment

Beta.18 adds `folder.openText({target, name})`. Both `folder.read` and the new
opt-in `folder.readText` permission are required; existing listing-only apps do
not gain content access. No new selection dialog is shown for each read. The
grant covers current immediate files in the selected directory, not only names
previously returned by the bounded listing, and is not a snapshot of file contents.

Names use the existing text-save basename grammar: at most 240 UTF-8 bytes,
without separators, ADS colons, controls, reserved device names, or trailing
dots/spaces. Reopen and verify the folder identity, then open only this basename
relative to the checked handle using Windows
[NtCreateFile](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/nf-ntifs-ntcreatefile)
with `RootDirectory`, `FILE_OPEN`, no reparse following and no offline recall.
Do not concatenate and reopen an ambient child path. A rename/replacement after
directory validation cannot redirect the relative child open to the replacement.
Reject child directories, reparse/offline/encrypted entries and files with more
than one hard link. A multiply-linked file may appear in the listing but cannot
be read through this method. Windows ACL checks still apply.

Reuse the existing bounded UTF-8 decoder: at most 2 MiB, optional BOM removed
from text but included in bytes, invalid UTF-8 and NUL rejected. Return the same
`{cancelled: false, name, text, bytes}` shape as `file.openText`, without any
path, child handle or write token. Missing/inaccessible files produce redacted
native failures; oversize and unsupported types use the existing file errors.
Reads share the folder operation queue and validate generation/revocation before
and after I/O. No retries, recursion, child-folder navigation or writes are added.

## Source Name-Bound Correction (2026-10-12)

The Beta.18 text-read step above reused `ValidateSaveName` and its
240-UTF-8-byte save-name bound, so `folder.openText` rejected listed valid
Windows basenames longer than 240 UTF-8 bytes even though the bounded listing
had returned them. Source now validates the read name with a separate
`ValidateFolderTextName`: at most 255 UTF-16 code units, where a supplementary
character counts as two; an initial UTF-8 bound of `utf8.UTFMax` x 255 = 1020
bytes; strict UTF-8 validity; and the same single-component basename grammar as
saves (no separators, ADS colons, controls, reserved device names, trailing
dots/spaces, or `.`/`..`). The text-save 240-byte rule (`ValidateSaveName`) is
unchanged.

The IPC `folder.openText` `name` field's schema `maxLength` is 255 Unicode
code points. That is a necessary bound only: the host additionally enforces the
255-UTF-16-unit limit, and the two bounds are not claimed equivalent. No new
permission, method or grant is added. Selected-directory identity, the
handle-relative `NtCreateFile` open, revocation, the
link/hardlink/ADS/traversal, offline and encrypted rejections, and the 2 MiB
content cap are all unchanged. Only a host built from this source gets the wider
bound; published alpha.68 and previously deployed hosts keep the 240-byte limit.
This is not limited to newly generated projects: an existing consumer repackaged
and run against the rebuilt host gains the same bound.

The Beta.18 history above retains its original 240-byte read bound.
Actual Windows tests create, list and read 255-unit ASCII, Hangul and emoji
basenames without changing disk bytes. The matched Go 1.27.1 windows/amd64
host builds measure 4,934,144 bytes each; startup checks pass. Exact receipts
and the distinction between native filesystem tests and mocked UI evidence
are recorded in `VALIDATION.md`.

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
Text-read tests cover explicit content permission, basename validation, deferred
completion, revocation during I/O, UTF-8/BOM/size rules, unchanged file bytes,
hard-link/reparse/offline rejection and directory replacement after validation.

## Rollback

Remove `folder.read` from a consuming manifest to disable the capability.
Removal of the methods/controller leaves the existing file APIs unchanged.
