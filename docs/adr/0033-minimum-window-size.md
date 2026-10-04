# ADR 0033: Minimum Window Size

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Decision

Add optional window.minWidth and window.minHeight to the manifest and packaged
runtime config, preserving contract version 1. Units are outer dimensions at
96 DPI, matching initial width/height. Zero/omitted means no app-specific bound
on that axis. Nonzero values must not exceed the corresponding initial dimension
or 16384 logical units. Centralize this validation across manifest, runtime
config and host adapter. Old bundles may reject these new optional fields;
consumers need a matching locally built/newly released CLI and host.

Only opting-in windows install a native subclass. Scale the configured bounds
for current DPI and cap them to the monitor work area. Apply minimum tracking
sizes after the existing window procedure has populated WM_GETMINMAXINFO.
Leave OS maximum size/position/tracking and unset axes unchanged.
See [WM_GETMINMAXINFO](https://learn.microsoft.com/en-us/windows/win32/winmsg/wm-getminmaxinfo).

Install after optional saved-placement restoration and before navigation.
Fit a normal window into its work area, enlarging it if below the effective
minimum. Protect normal resize/restore through WM_SIZE and adjust suggested
normal rectangles during WM_DPICHANGED using the message's new DPI and target
monitor. Guard synchronous SetWindowPos re-entry. Do not activate/show windows
or change maximized sizing; minimize/maximize continue to use Windows behavior.
Transient monitor-query failure retains normal Windows sizing for that event.
An initial installation failure returns a startup error and removes its state.

## Scope and Cost

This is a manifest option, not a script-controlled resize or new IPC permission.
Small monitors may necessarily provide less room than the configured logical
minimum. Responsive layout and scrolling remain app responsibilities.
No host timer, worker, filesystem watcher, dependency or DB/state-format change.
One small owner record exists only per opted-in window and is removed on
WM_NCDESTROY. Native queries run only during initial fit and relevant window
events. Measure same-toolchain host growth; do not infer startup speed.

## Verification and Rollback

Cover zero/one-axis defaults, bound/type rejection, config round-trip and
packaged values; arithmetic at 96/120/144 DPI; smaller and negative-coordinate
work areas; native tracking, programmatic normal resize, suggested DPI rects,
unchanged unset/max geometry, no show/activation and destroy/failure cleanup.
Physical monitor/DPI drag and visual resize remain separate manual evidence.
Remove both minimum fields to return to normal Windows sizing. Existing saved
placement is kept; no profile migration or deletion is needed.
