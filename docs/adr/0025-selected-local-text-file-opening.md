# ADR 0025: Selected local text-file opening

- Status: Accepted
- Date: 2026-10-02
- Owner: Project maintainer
- Amends: ADR 0017 (one selected-file read capability)

## Decision

Add the opt-in permission `file.open` and method `file.openText`, accepting only
an empty parameter object. Resolve its promise after a host-owned Windows
IFileOpenDialog completes, not from inside the WebView event callback. The
default permission list remains empty. There is no arbitrary path parameter,
filesystem enumeration, write, process, reusable token, or persisted grant.

Read one selected local disk file, at most 2 MiB. Keep one read-only native file
handle for validation and bounded reading, then close it before returning only
`cancelled`, `name`, `text` and `bytes`. Refuse UNC/device/ADS paths, directories,
final-component reparse points, offline files, invalid UTF-8 and NUL bytes.
Check the opened handle's final path so a directory junction cannot redirect
reading to a network share. Local parent junctions and hard links are allowed;
this is explicit file selection, not directory sandboxing. A same-user actor
that can replace the selected path before it is opened remains outside the
host's protection claim. A UTF-8 BOM is stripped only from returned text.

## Lifecycle and Limits

Only one selection may be pending. Reserve the logical IPC request ID until
completion, reject duplicate requests, and map cancellation to a successful
`cancelled: true` result without reading. Recheck document generation before
showing the dialog and before reading. Transport promises carry a per-document
session for response correlation, not authentication. Native top-level origin
and manifest permission checks remain authoritative. Shutdown discards responses.

Request limits remain 64 KiB; selected text responses can be larger, bounded
by the 2 MiB file limit (JSON escaping can expand text up to about 12 MiB).
Reading is synchronous and bounded, on demand, on the existing UI thread.
There is no worker, timer, background reader, new dependency, network request,
or change to the browser File System Access API. Lazy COM/DLL objects are used
only for opted-in calls. Binary size still grows and must be measured; no idle
CPU or latency improvement is claimed.

## Validation and Rollback

Test permission/parameter denial, duplicate IDs, cancellation, busy release,
navigation/shutdown, UTF-8/BOM and exact/oversized bounds, read-only disk
contents and path/placeholder exclusions. Transport tests cover deferred
execution, document correlation and one-shot completion. Build the read-only
Text Viewer example and inspect its permissions. Real dialog selection remains
a distinct manual check unless actually exercised.

Remove `file.open` to disable the method. Existing File Notes continues using
its browser file API. Native saving and File Notes migration are separate work.

## References

- [WebView2 threading and reentrancy](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/threading-model)
- [IFileOpenDialog](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nn-shobjidl_core-ifileopendialog)
