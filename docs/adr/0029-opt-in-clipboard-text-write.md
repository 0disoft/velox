# ADR 0029: Opt-In Clipboard Text Write

- Status: Accepted
- Date: 2026-10-03
- Owner: Runtime host

## Decision

Add only `clipboard.writeText({text})`, gated by the independent manifest
permission `clipboard.write`. Existing apps gain no clipboard access. Success
returns `null` after writing `CF_UNICODETEXT`; no text, path or native detail is
returned or logged. There is no read, format enumeration, listener, history,
automatic copy, persistent grant, timer, worker, or new dependency.

Accept a string of at most 32 KiB of UTF-8, including empty text. Reject embedded
NUL and invalid UTF-8. The unchanged 64 KiB serialized IPC limit also applies,
so heavily escaped text can reach the wire limit before the text-byte limit.
Unknown, duplicate and non-string parameters are rejected before native access.

Execute on the existing UI thread with the app window as clipboard owner.
Prepare a movable, NUL-terminated UTF-16 allocation before opening or emptying
the clipboard. Free it on failure; transfer ownership to Windows only after
successful `SetClipboardData`. Always close an opened clipboard. If another
process holds it, return `CLIPBOARD_BUSY` without clearing or retrying.
Other native failures are redacted. A failure after `EmptyClipboard` can leave
the clipboard empty; this is not a transactional replacement guarantee.

The implementation follows Microsoft's [clipboard ownership contract](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setclipboarddata)
and [window-owner requirement](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-openclipboard).

## Trust Boundary

The existing top-level origin, denied-frame, request and shutdown checks remain
authoritative. IPC does not attest browser user activation. Granting this
permission lets trusted app scripts replace clipboard contents without a host
confirmation, including without a click; consumers should invoke it only from
an explicit copy action. The permission is not a defense against compromised
assets of an opted-in app. Clipboard history or cross-device synchronization
is controlled by Windows, not Velox. Applications should not copy secrets
automatically. Browser-native selection copy/paste is unchanged.

## Cost and Verification

Disabled apps allocate no clipboard writer. Enabled apps do local work only on
invocation, without monitoring, sleeping or polling. Tests use fake Win32 calls
to verify Unicode termination, ownership transfer, cleanup and failure ordering
without reading or replacing the maintainer's clipboard. Real copy/paste remains
a separate manual check. Measure host executable growth with matching toolchain
and build flags; do not claim zero size or startup impact.

## Rollback

Remove `clipboard.write` from the app manifest to disable this capability.
The method and writer can be removed without changing existing file/folder APIs.
