# ADR 0034: User-Controlled Always On Top

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Decision

Add optional boolean `window.alwaysOnTop` to manifest/runtime config v1.
The default is false. Apply true after window placement/minimum sizing and
before navigation. All app windows expose a checkable "Always on top" system
menu item, accessible through Alt+Space or the title-bar system menu. This
lets users enable or undo topmost without relying on application JavaScript.
Old CLI/host bundles may reject the new field; use a matching new bundle.

Use HWND_TOPMOST/HWND_NOTOPMOST with SWP_NOMOVE, SWP_NOSIZE and SWP_NOACTIVATE.
Toggling does not show, restore, resize or activate the window. Other topmost
windows still share the topmost band; this is not an exclusive overlay.
Owned dialogs follow Windows z-order rules.
See [SetWindowPos](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwindowpos).

Read WS_EX_TOPMOST for toggles and refresh check state when a menu opens.
Do not cache a second state or serialize it into saved placement. Restart
reapplies the manifest default, even with rememberState enabled. Installation
failure rolls back the menu/subclass and fails startup. Toggle failure keeps
the actual OS state and reports a native diagnostic. Remove the subclass on
WM_NCDESTROY. No public IPC method, permission, timer, worker, dependency,
filesystem operation or state migration is added.

## Verification

Check boolean/default parsing, manifest/runtime round-trip, native initial
state, menu toggling/readback, unchanged geometry/visibility/focus, external
native state synchronization, invalid handles, destruction and fresh-window
defaults. Resolve SetWindowPos before installing the callback, including when
initial topmost is false. Native tests initially failed when its first lazy
lookup occurred inside the toggle callback; pre-resolution passed repeated
default-off-first tests. An earlier non-inlining hypothesis did not solve it.
Validate subclass identity/reference data before handling commands.
Measure matched-toolchain host size separately from performance. Actual
visible overlap, Alt+Space interaction and dialogs remain manual checks.
