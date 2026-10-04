# ADR 0032: Bounded Window Attention

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Decision

Add `window.requestAttention({count})` and `window.cancelAttention({})` under
independent, default-disabled `window.attention`. Do not grant it through
window.basic or window.title. Count is an optional JSON integer from 1 to 5,
default 3. Cancel takes an empty object. Strict fields and existing shutdown,
origin and wire/in-flight checks apply.

Use FlashWindowEx with FLASHW_TRAY, bounded uCount and default blink rate.
Never set continuous timer flags or flash the caption. Skip a request for
the foreground window; explicit cancel uses FLASHW_STOP. Do not call
ShowWindow, SetForegroundWindow or restore/show hidden-to-tray windows.
Destroying the window releases its native flashing state.

FLASHWINFO uses native ABI alignment. The API's BOOL reports prior active
state, not success. Check HWND validity separately and redact adapter errors.
See [FlashWindowEx](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-flashwindowex)
and [FLASHWINFO](https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-flashwinfo).

## Scope and Cost

This is a visual attention hint, not a toast or a persistent notification.
Windows shell/accessibility settings may suppress/change it. A hidden window
may lack a taskbar button. Success cannot attest that a person saw the cue.
Trusted scripts can request it without browser activation and repeated calls
can restart the sequence; count is a per-request bound, not a rate limit.

No host timer, worker, watcher, stored grant, dependency or DB change is added.
An isolated example may use one cancellable frontend timeout to let a tester
switch windows before the native request. File Notes remains unchanged.
Measure host growth with matching Go/build flags.

## Verification and Rollback

Test permission independence and manifest/runtime/build-report propagation,
default/bounded counts, strict cancel, shutdown, redacted failures, native
structure layout, no continuous flags and no visibility/foreground mutation.
Native tests use a disposable hidden window and do not prove visible taskbar
flashing; the example's request/cancel behavior requires manual observation.
Remove window.attention to disable both methods without migration.
