# ADR 0030: Confirmed Clipboard Text Read

- Status: Accepted
- Date: 2026-10-03
- Owner: Runtime host

## Decision

Add only `clipboard.readText({})` under the independent, default-disabled
`clipboard.read` permission. ADR 0029's write permission never grants reading.
Before clipboard access, display a host-owned Yes/No confirmation naming the
app and warning about private information. No is the default. Approval applies
only to that request, not future reads. The dialog contains no clipboard data.

Return `{cancelled: true}` on refusal, without opening the clipboard. On
approval return `{cancelled: false, text}`; empty Unicode text is valid.
Request only `CF_UNICODETEXT`, including Windows-synthesized Unicode text.
Reject unsupported formats and malformed UTF-16. Limit decoded text to 32 KiB
of UTF-8, without embedded NUL. The native NUL terminator ends the text; trailing
allocation padding is not text. Bound copies to 32,769 UTF-16 code units rather
than allocate from an untrusted/unbounded clipboard size.

Use the existing deferred UI dispatch and reserve the IPC request ID until
completion. Allow only one pending read confirmation per host. Check document
generation and shutdown before confirmation, after confirmation and after
reading; stale work returns no text. Never retry busy access automatically.
No listener, history, timer, worker, persistent grant or dependency is added.
No formats other than Unicode text are exposed and no clipboard contents are
written, emptied, logged or retained by the reader after completion.

## Ownership and Errors

Clipboard handles remain owned by Windows. Copy bounded memory while locked,
then unlock before closing the clipboard; never free the borrowed handle.
This follows [GetClipboardData](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getclipboarddata)
and [GlobalSize](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-globalsize).
Allocation size may include padding, so it is not the decoded text length.

Missing permission is `PERMISSION_DENIED`; nonempty parameters are
`INVALID_PARAMS`; a queued read is `TOO_MANY_REQUESTS`; clipboard contention is
`CLIPBOARD_BUSY`; unsupported or invalid Unicode is `UNSUPPORTED_TEXT`;
oversized text is `PAYLOAD_TOO_LARGE`. Native failures and stale-document work
return redacted `NATIVE_OPERATION_FAILED`; shutdown returns `SHUTTING_DOWN`.
Close failure also discards text rather than returning a successful result.

## Trust and Cost

Existing top-level origin and denied-frame boundaries remain authoritative.
IPC does not attest browser user activation: trusted scripts may request a
dialog, but cannot bypass per-request native approval. Once approved, returned
text is available to the app's scripts; the permission cannot prevent those
scripts from storing or transmitting it. Consumers should invoke only from an
explicit paste action and avoid automatic persistence of private contents.
Browser-native copy/paste remains unchanged.

Clipboard text is read at the moment after approval, not snapshotted when the
request starts. Other applications can change it while the prompt is open.
Windows clipboard retrieval and delayed rendering can block the UI thread;
there is no bounded-latency guarantee. Measure matching-toolchain host growth,
and keep manual confirmation separate from fake-native tests.

## Verification and Rollback

Tests cover default denial, strict parameters, deferred/single completion,
refusal without access, Unicode/size bounds, lock cleanup, contention, and
navigation/shutdown without text disclosure. Native allocation readback tests
do not touch the user's clipboard. Actual approval and paste remain manual.
Remove `clipboard.read` to disable the feature without affecting write or file
permissions. No DB or stored-grant migration is required.
