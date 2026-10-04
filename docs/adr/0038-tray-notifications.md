# ADR 0038: Tray Notification Balloons

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host
- Amends: ADR 0024 (host-owned tray)

## Context and Decision

Opted-in tray apps need a short, user-visible status message without a second
process, a toast registration, or background delivery. Add one independent,
default-disabled `notification.show` permission and one IPC v1 method,
`notification.show({kind, message})`. `window.tray` neither grants nor
requires it, and it grants no other window or tray operation.

- `kind` is required and limited to `info`, `warning`, or `error`; it
  selects the balloon icon.
- `message` is required text with at least one non-whitespace character, at
  most 255 UTF-16 code units, and no NUL, DEL or other C0/C1 control character
  except line feed (U+000A) and tab (U+0009).
- Only those two fields are accepted. Unknown fields, missing or wrong-typed
  values, whitespace-only messages, oversized text, and disallowed controls
  return `INVALID_PARAMS`.

A successful result is `null`, which means the Windows shell accepted the
request; it is not proof that a person saw the balloon.

## Native Behavior

Require an opted-in, live tray: the manifest must set `window.tray: true` and
the host's notification-area icon must be currently registered. Otherwise
return redacted `NATIVE_OPERATION_FAILED` and install no icon; this method
never creates or recovers a tray on its own.

Reuse the registered icon and call `Shell_NotifyIconW` with `NIM_MODIFY` and
a transient local `NOTIFYICONDATAW` copy carrying `NIF_INFO | NIF_REALTIME`.
`NIF_REALTIME` makes the shell drop rather than queue a balloon it cannot show
immediately. Use the manifest application name as the fixed `szInfoTitle`,
Unicode-safe truncated to 63 UTF-16 code units (the field holds 64 including
the terminator). Copy the validated message into `szInfo` and set
`NIIF_NOSOUND | NIIF_RESPECT_QUIET_TIME` together with the `info`,
`warning`, or `error` value in `dwInfoFlags`.

Keep no stored body and do not replay a balloon after an Explorer restart; only
a fresh `notification.show` call may show another balloon. The host allocates
and discards the structure per call, so nothing is persisted.

When the shell reports `NIN_BALLOONUSERCLICK` (0x405) for the icon, reveal the
existing window exactly as the tray menu's Open window command does. That
respects the existing modal-owner and shutdown gates and never creates a second
window.

## Scope and Alternatives

A full Windows toast, notification history, scheduling, action buttons, or a
delivery callback API is deferred beyond this transient-balloon scope.
Windows 10/11 settings can suppress balloons, so
acceptance is best-effort presentation, not guaranteed display. The balloon
belongs to the opted-in tray icon and cannot outlive it.

This adds one permission, one method, one `NOTIFYICONDATAW` fill and one click
branch. It adds no dependency, timer, worker, background process, polling, file,
DB or state-format change, and no version change. File Notes and the default
capability defaults stay unchanged.

## Verification and Rollback

Required coverage: independent permission and packaged propagation, strict
two-field grammar including kind, whitespace, UTF-16 length and control limits,
success `null` semantics, missing/disabled tray failure without installation,
title truncation, native `NIM_MODIFY` with `NIF_INFO | NIF_REALTIME`, no
body retention across restart, and `NIN_BALLOONUSERCLICK` reveal through the
normal modal/shutdown gates. Actual balloon visibility and Windows quiet-time or
disabled-notification behavior remain shell-controlled manual evidence; no
validation result is claimed here. Remove `notification.show` to disable the
method without migration, or set `window.tray` to `false`.

## Revisit Triggers

- A request for persistent history, scheduling, action buttons, or delivery
  confirmation.
- A need to show notifications without an opted-in tray icon.

## Synchronized Surfaces

IPC v1, CLI configuration, product specification, ADR index, and validation
requirements.

## References

- [NOTIFYICONDATAW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-notifyicondataw)
- [Shell_NotifyIconW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shell_notifyiconw)
