# ADR 0035: Optional Fixed Window Size

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Decision

Add optional boolean `window.resizable` to manifest/runtime config v1.
Omitted values retain true and preserve old bundle config bytes. Explicit
false disables user resizing and maximization; minimize, move, close and the
host system-menu entries remain available. Use a matching new CLI/host bundle;
older versions may reject the new field.

For false only, remove WS_THICKFRAME and WS_MAXIMIZEBOX and refresh the frame
with SWP_FRAMECHANGED without moving, resizing, showing or activating it.
Block SC_SIZE and SC_MAXIMIZE in a small window subclass. The existing
window.maximize IPC method also fails with redacted NATIVE_OPERATION_FAILED;
window.basic does not override this app policy. Restore/minimize keep their
normal behavior. This is a user-facing sizing policy, not a security boundary
against other desktop processes or an arbitrary native SetWindowPos call.
See [window styles](https://learn.microsoft.com/en-us/windows/win32/winmsg/window-styles)
and [SetWindowPos](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwindowpos).

With rememberState, restore saved position but use current configured outer
width/height at the target monitor DPI, capped to its work area. Ignore saved
maximization. Keep the existing state format and saved-position scaling.
Ordinary resizable windows keep their previous full-placement restoration.
Existing minimum-size handling and WebView2 DPI suggested-rectangle handling
continue; fixed sizing does not intercept DPI-change geometry or programmatic
resize. No new persistent setting or migration is introduced.

## Cost and Verification

Default/true installs no fixed-size subclass or native style work. False adds
one callback, no owner map, timer, worker, dependency or new IPC permission.
Remove its subclass on destruction. Installation errors fail startup and
roll back edited style/handler where applicable.

Verify omission/true/false serialization, invalid types, default style retention,
fixed frame bits, system commands, IPC maximize rejection, unchanged initial
geometry/visibility/focus, invalid handles, destruction, saved position with
configured size, ignored saved maximization, DPI arithmetic and small work
areas. Physical monitor dragging, keyboard snapping and frame interactions
remain manual evidence. Measure host size; do not infer latency from it.
