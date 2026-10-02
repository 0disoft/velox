# ADR 0024: Opt-in system tray

- Status: Accepted
- Date: 2026-10-02
- Owner: Project maintainer
- Amends: ADR 0017 (narrow host-owned lifecycle amendment)

## Context

Small Windows utilities often need an explicit way to hide and reopen their one
window. Velox must preserve its compiler-free consumer build, closed IPC table,
and low idle overhead. Automatically converting close into hide could strand
unsaved edits or silently leave an application running.

## Decision

Add `window.tray`, an optional boolean with a `false` default. Only opted-in
hosts install a window subclass and register a notification-area icon, reusing
the window's shared icon and a bounded UTF-16 application-name tooltip.

- Use the Windows notification-area version-4 mouse and keyboard events.
- Selecting the icon reveals/restores the window and requests foreground.
- A fixed native menu offers Open window, Hide window, and Quit. No app-defined
  menus, callbacks, notification messages, or icon path configuration are added.
- Hide is explicit and allowed only while the icon is registered and the owner
  is enabled. It does not replace X, Alt+F4, or web window-close behavior.
- Quit reveals the owner and posts normal `WM_CLOSE`; it does not destroy the
  WebView directly or bypass `beforeunload`. Cancellation retains the icon.
- Recover through `TaskbarCreated` after Explorer restart, without polling. If
  registration fails, reveal the window and refuse further tray hiding. A later
  restart may retry. Initial registration failure fails host startup.
- Normal destruction deletes the shell icon and removes the subclass owner.
  Windows owns the shared HICON; the host does not destroy it. Forced termination
  is not guaranteed to delete the shell's stale icon immediately.
- Single-instance activation also reveals a hidden window. Foreground remains
  best-effort under Windows focus policy.

No extra dependency, goroutine, timer, server, process, public IPC method,
permission, or consumer compilation step is added. A hidden WebView continues
running; background CPU or memory savings are not claimed. The default path
does not initialize the tray callback, register the restart message, or install
this subclass. Native code still increases the shared binary size; measure it.

## Scope and Threat Model

The menu is host-owned and carries only fixed lifecycle commands. No untrusted
menu text, URL, file, command-line payload, or arbitrary code crosses it. Tray
events are not authentication: a same-user process can send window messages.
Modal-owner disabling and closing checks avoid normal UI reentrancy; they are
not a hostile-process boundary. ADR 0017's backend, process, sidecar, plugin,
filesystem, updater, and broad native-API restrictions otherwise remain.

## Alternatives

- Automatic close-to-tray: rejected because it changes exit and unsaved-edit
  expectations. Use the explicit Hide window command.
- General tray/menu framework: deferred because it adds public API and dependency
  surface for three host-owned commands.
- Poll for Explorer or keep an extra hidden owner process: rejected because the
  existing window message loop and restart broadcast are sufficient.
- Enable by default: rejected because most one-window apps do not need a tray.

## Consequences

The feature keeps one event-driven host and gives utility apps a recoverable
hidden-window path. It expands configuration and executable size, keeps hidden
web content alive, depends on the Windows shell for icon visibility, and cannot
guarantee foreground or immediate stale-icon cleanup after forced termination.
Windows may place the icon in its overflow area.

## Validation Plan

- Default/true/false and wrong-type manifest/runtime parsing and round trips.
- Native show/hide/minimize/maximize behavior, version-4 event identity,
  registration/version failure rollback, restart success/failure, modal-owner
  checks, closing suppression, and normal destruction cleanup.
- Cancelled normal close retains the icon; Quit posts normal `WM_CLOSE` and
  reveals its owner before confirmation. Real browser confirmation remains a
  manual behavior check, not inferred from disposable native-window tests.
- Hidden single-instance window activation and unchanged default startup smoke.
- Shared host size comparison against the pre-tray build. Idle CPU comparisons
  and live shell-menu/Explorer-restart checks are not implied by source tests.

## Rollback and Revisit

Remove `tray` or set it to `false`; no profile migration or cleanup file is
required. Revisit for dynamic menus, notifications, automatic close-to-tray,
platform expansion, or any measured performance claim.

## Synchronized Surfaces

Manifest/runtime structs and schemas, host configuration, native window helpers,
single-instance activation, File Notes source configuration, configuration and
product docs, and ADR index. Changing source does not update existing packaged
executables or authorize publication.

## References

- [Shell_NotifyIconW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shell_notifyiconw)
- [NOTIFYICONDATAW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-notifyicondataw)
- [TrackPopupMenu](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-trackpopupmenu)
