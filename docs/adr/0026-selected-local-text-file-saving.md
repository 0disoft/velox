# ADR 0026: Selected local text-file saving

- Status: Accepted
- Date: 2026-10-03
- Owner: Project maintainer
- Amends: ADR 0017 (one selected-file write capability)

## Decision

Add opt-in `file.save` and the frozen JavaScript helper
`window.velox.saveText(text, name = "Untitled.txt")`. Every save shows a
host-owned IFileSaveDialog outside the WebView callback. Its overwrite prompt
is enabled. Applications supply only UTF-8 text and a suggested base filename;
they cannot supply a path. Return `{cancelled, name, bytes}`, never a path or
reusable filesystem grant. Cancellation writes nothing and consumes the upload.
No save-to-last-path or persisted grant is added. File Notes migration is separate.
ADR 0027 adds separate, explicit connected-save methods; this original per-call
selection contract remains unchanged for `saveText` and `file.commitSave`.
The dialog offers Text documents (`*.txt;*.text`), Markdown (`*.md;*.markdown`)
and All files (`*.*`). Choose the initial filter from the suggested extension;
names without an extension default to text. Windows updates the default extension
when the user changes the file type. These choices do not convert the UTF-8 body
into another document format.

## Bounded Transport and Lifecycle

Retain 64 KiB per native request and WebView message. The helper sends at most
2 MiB of UTF-8 text through `file.beginSave`, ordered `file.appendSave` calls,
then deferred `file.commitSave`. `file.cancelSave` discards abandoned text.
The helper preserves surrogate pairs and sends at most 4,096 UTF-16 code units
per chunk, leaving room for worst-case JSON escaping and the binding envelope.
NUL and malformed helper input are rejected before admission. Native chunks
must also be valid UTF-8 without NUL. Byte offsets and the declared final byte
count are checked; incomplete, overrun, unknown-field and stale-token requests
cannot reach the dialog. The upload token identifies text, not a file grant.

Keep only one bounded upload or pending save per host. Allocate its buffer on
demand. Drop staged text on accepted navigation and shutdown; never reuse its
token within the host lifetime. Once commit consumes the upload, check document
generation before showing the dialog and again before writing. The commit ID
remains reserved until completion; per-document transport correlation prevents
delivery to a replacement document. Dialogs share the existing UI thread and
disabled-owner check with file opening. No worker, timer, network client or
dependency is added. Text staging and synchronous disk I/O are on-demand costs,
not a claim of improved idle CPU, latency or zero memory overhead.

## Filesystem Boundary and Recovery

Accept canonical absolute local-drive paths from native selection only. Refuse
UNC/device/ADS paths, remote drives, reserved filenames, linked parent or final
components, directories, offline, readonly, encrypted and multiply-linked
existing files. A missing parent fails; do not create directories.

Write a sibling temporary file, flush and close it before committing. When
replacing an existing file, apply its DACL to the temporary file before writing
the text, hold a read handle denying shared writes, and recheck file identity
and metadata immediately before replacement. Use ReplaceFileW without ignoring
ACL/merge failures and with a unique sibling backup. Delete the backup only
after replacement succeeds. For a new file, MoveFileExW has no replacement
flag, so a newly appearing target is not silently overwritten.

If replacement or backup cleanup fails, return `SAVE_RECOVERY_REQUIRED` and
retain recovery files in the selected folder. This can mean the target already
contains the new text; callers must not claim success or discard their editor
buffer. Recovery may require examining `.velox-save-*` and `.velox-backup-*`
there. Do not claim that all Windows replacement failures leave the original
name intact. A crash can leave these files too. This is not a power-loss-proof
transaction or protection against a same-user actor replacing paths between
validation and native filesystem calls. The shell dialog may browse locations
under OS policy; unsupported final selections are rejected before text writing.

## Validation and Rollback

Cover real disk creation/replacement/readback, empty/Unicode/exact 2 MiB text,
restricted DACL preservation, invalid text, readonly/locked/link exclusions,
deferred cancellation, navigation, busy release, upload limits, byte offsets,
duplicate completion, shutdown, stable private errors and helper serialization.
Build/inspect the `examples/file-saver` package and measure host growth.
Real Save as, overwrite confirmation and cancellation remain manual checks
until a maintainer actually exercises them. Remove `file.save` to disable all
four wire methods; existing browser File Notes remains unchanged.

## References

- [IFileSaveDialog](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nn-shobjidl_core-ifilesavedialog)
- [File dialog options](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/ne-shobjidl_core-_fileopendialogoptions)
- [ReplaceFileW behavior and recovery](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-replacefilew)
