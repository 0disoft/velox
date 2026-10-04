# ADR 0039: Activation Shortcut

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Context and Decision

A hidden, minimized, or background window can be hard to bring back without a
tray icon. Add one optional, default-disabled manifest and runtime-config
string, `window.activationShortcut`, that registers a single system-wide hot
key to reveal the existing window. Omission or an empty string disables the
feature and installs nothing.

The value must be exactly `Ctrl+Alt+<key>` or `Ctrl+Alt+Shift+<key>`, where
`<key>` is one uppercase ASCII letter `A`-`Z` or one digit `0`-`9`.
Rejecting `null`, numbers, booleans, arrays, and objects keeps the field
string-only. Within a string, reject any other modifier order or set (for
example `Alt+Ctrl` or `Win`), extra or duplicated modifiers, lowercase
letters, the multi-character key name `F12`, embedded spaces, an empty key, multibyte
characters, and trailing text.

This adds no permission, no IPC method or event, no key logging, no general
global-key registry, no custom actions, and no in-app shortcut configuration
UI. It cannot observe arbitrary keystrokes; it reacts only to its one combo.
The optional tray (ADR 0024) is neither required nor implied: a tray-less app
can still be revealed by the shortcut when the field is set.

## Native Behavior

Register exactly one binding on the existing UI thread with
[RegisterHotKey](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-registerhotkey)
against the window's existing HWND, using the parsed modifiers, the key, and
`MOD_NOREPEAT` so a held combo does not repeat. The system posts
[WM_HOTKEY](https://learn.microsoft.com/en-us/windows/win32/inputdev/wm-hotkey)
to that HWND; a host-owned subclass handler validates the registration id, the
modifier bits, and the virtual key before acting, then calls `DefSubclassProc`
for every other message.

On a validated message the handler reveals the existing window: show it
without changing maximized placement, restore it only if minimized, and request
foreground as a best-effort. A maximized window keeps its maximized placement.
The check also respects the current enabled state and the dispatcher's closing
flag, so a disabled modal owner or a shutting-down host does nothing.

Referencing the debugger-reserved `F12` and the OS-reserved Windows key is
rejected during parsing, so no reserved combo is ever registered.

## Failure Handling

`RegisterHotKey` fails when another process already owns the same system-wide
combo, and that is a normal outcome, not a fatal error. Registration never
overrides or steals another process's binding. If registration fails after the
host installed its subclass, the host rolls back only its own subclass and
internal map entry, leaves any other process's binding untouched, and never
calls `UnregisterHotKey` for a binding it does not own. Any other install
failure takes the same path.

Every failure emits one host-owned warning that the shortcut is unavailable.
The application then continues normally with the shortcut disabled; there are
no retries, no polling, and no fallback registration.

## Lifetime

On successful registration the host explicitly calls
[UnregisterHotKey](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-unregisterhotkey)
exactly once, on `WM_DESTROY` or `WM_NCDESTROY`, whichever arrives, guarded so
a second message is a no-op. A cancelled `WM_CLOSE` therefore keeps the
binding until the window actually closes. The minimal reveal helper is shared
with the tray path, preserving its behavior. The single-instance path is unchanged.

## Scope and Alternatives

The shortcut is a fixed reveal action, not a scripting hook: there is no IPC
permission, method, or event, no key logging, no general global-key API, no
custom action dispatch, and no configuration UI. Shortcut conflicts are the
OS's normal system-wide behavior, so two separate applications, two profiles,
or two instances of one app that use the same combo collide even though their
JSON configuration is independent; a per-app lock cannot arbitrate a
system-wide combo. The existing optional `app.singleInstance` flag (ADR 0022)
can avoid the same-app multi-instance case but is not required here.

This adds one manifest/runtime string, one parser, one subclass handler, and
one registration. With the field omitted or empty, the host registers nothing
and installs no subclass. A registered binding is not re-created or polled on
Explorer restart or any system settings change. No timer, worker, background
process, dependency, file, DB or profile-format change, registry write, or
version bump is added, and File Notes and the default capability set are
unchanged. Prefer `Ctrl+Alt+Shift+<key>` in examples because some layouts use
`Ctrl+Alt` (AltGr) for typed characters.

## Verification and Rollback

Required coverage: focused unit tests for the grammar (accepted canonical
forms; rejected null and non-strings, modifier order, spaces, lowercase,
duplicates, `Win`, `F12`, multibyte and empty keys) and native-HWND mock tests
that drive `WM_HOTKEY` with valid and invalid id/modifier/key values, mock
registration-collision and release behavior, `MOD_NOREPEAT`, rollback without
unregistering an unowned binding, exactly-once unregister across
`WM_DESTROY`/`WM_NCDESTROY`, and no native registration or subclass when the
field is absent or empty. Separate manual checks must confirm that a physical
key press reveals a hidden, minimized, and background window and preserves a
maximized window. No completed validation is claimed here.

Remove `window.activationShortcut` to disable the feature with no migration.

## Revisit Triggers

- A request for more keys, multiple shortcuts, key remapping, or a
  configuration UI.
- A request to observe keys other than the single reveal combo.

## Synchronized Surfaces

CLI configuration, product specification, ADR index, and validation
requirements.

## References

- [RegisterHotKey](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-registerhotkey)
- [UnregisterHotKey](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-unregisterhotkey)
- [WM_HOTKEY](https://learn.microsoft.com/en-us/windows/win32/inputdev/wm-hotkey)
