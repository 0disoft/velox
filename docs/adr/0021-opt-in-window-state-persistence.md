# ADR 0021: Opt-in window state persistence

- Status: Accepted
- Date: 2026-10-02
- Owner: Project maintainer
- Amends: ADR 0017 (narrow window lifecycle amendment)

## Context

ADR 0017 kept the host's native surface closed to basic window lifecycle and
required a new product and threat-model ADR before any wider native capability.
ADR 0020 amended that boundary for compiler-free branding and a per-user
installer only; it did not approve runtime window behavior beyond the existing
lifecycle.

A packaged Velox application currently opens at its manifest `window.width` and
`window.height` every time. Users who resize or maximize a window lose that
placement on the next launch, a small but real usability gap for the desktop
apps the product targets. Restoring the previous placement needs no new IPC
method, no application-native compilation, and no new Windows permission set:
the host already owns exactly one top-level window and already creates a
per-application WebView2 profile directory.

This ADR approves only the narrow window-state persistence described below. It
does not approve a broader native API, a general settings store, or any
application-visible native surface.

## Decision

Add one optional manifest field, `window.rememberState`.

- Type: boolean. Default `false`; omitted values behave as `false`.
- When `false` or omitted, the host performs no window-state file I/O and
  installs no window subclass. Startup, sizing, and shutdown behavior are
  unchanged.

When `window.rememberState` is `true`, the shared host persists one bounded
window-state record per application:

- Location: `velox-window-state.json` under the application's profile
  directory, which is `VELOX_DATA_DIR` when set, or the default
  `%LOCALAPPDATA%\Velox\profiles\<app-id>` profile otherwise. This is the
  same per-application profile the WebView2 user data already uses.
- Record contents: the raw physical screen normal window rectangle, the work
  area of the monitor that owned the window, the window DPI, the maximized
  flag, the application ID, and a state-format version, currently `1`.
- The state-format version is independent of `app.version`. A valid record is
  accepted across application updates, so shipping a new application version
  does not discard the saved placement.
- On startup the host loads at most 4096 bytes. A missing, malformed, oversized,
  wrong-application, or wrong-state-format-version record is ignored, and the
  manifest size applies instead.
- A valid record is restored after the window is created but before entry
  navigation and before the interactive app is ready. The host scales the
  stored rectangle for the current DPI, clamps it into the current display
  work area, and enforces a minimum normal size of 320 by 240 logical units.
- A minimized close is never restored as minimized. The next launch opens a
  visible normal window, or the previously maximized state when that was the
  last state.
- The record is written once during a normal `WM_DESTROY` window destroy. A
  cancelled close does not write. Forced termination, a crash, or an
  externally killed process is not guaranteed to persist anything.

The host adds no timer, polling loop, background thread, scheduled task,
sidecar, marshalled helper, or new dependency. It adds no IPC method, no
filesystem permission, no shell or process capability, and no new native
surface visible to web content. The state file is host-owned operational data,
not an application API.

Persisting state writes to the per-application profile directory Velox already
creates. It does not change the portable output, the deterministic ZIP, the
release bundle, or the installer payload.

### Amendment to ADR 0017

ADR 0017 required a new product and threat-model ADR before any native
capability beyond basic window lifecycle. This ADR is that review, and it is
deliberately narrow: it approves opt-in window-state persistence only. The
application-specific backend, native filesystem, shell, process, sidecar,
plugin, local-server, updater, asset-sealing, new-platform, and broad IPC
prohibitions in ADR 0017 remain in force and are unchanged.

## Alternatives

### Keep the manifest size on every launch

Retained as the default behavior. It is the cheapest and most predictable
option and stays the behavior whenever `rememberState` is absent.

### Store state through a new IPC method or native settings API

Rejected. It would widen the closed IPC v1 method table and give web content a
native surface for a purely host-owned convenience, breaking the narrow
boundary for no added value.

### Restore with a subview or a background watcher

Rejected. The host already receives `WM_DESTROY`, so a one-shot write needs no
timer, poll, or background process, and window state needs no continuous
observation.

### Add a general-purpose persistent settings store

Rejected. A general store would invite application state, migrations, and
versioning far beyond restoring one window, and would blur the browser-owned
storage boundary.

### Let the application restore its own window size

Rejected. Static assets cannot resize or position the native top-level window,
and adding that capability would create exactly the wider native API this
decision avoids.

## Consequences

### Positive

- Users keep their window size, position, and maximized state across launches
  when the application opts in.
- The default path adds no file I/O, no subclass, and no behavior change.
- The feature reuses the existing profile directory and needs no new
  dependency, IPC method, permission, or background process.
- The stored record is bounded and self-identifying, so stale or foreign files
  are ignored rather than trusted.

### Negative

- The host gains a small Windows window-procedure and file-I/O path that needs
  tests for malformed input, DPI changes, and work-area clamping.
- The state file is host-owned local data on disk; display changes force
  conservative fallback behavior.
- A record can be lost on a forced kill or crash, and the behavior must stay
  honest about that limitation.
- One more opt-in field widens the manifest surface and must stay synchronized
  with the schema, the parser, and the docs.

## Validation

An accepted decision is not proof that tests pass. The host, manifest schema,
runtime-config schema, parser, and builder support now exist; these checks are
the acceptance verification for the feature:

- Schema and parser accept `window.rememberState` as an optional boolean with a
  `false` default, and reject a non-boolean value.
- A record written by one application version is accepted by a later version of
  the same application, because validity is keyed on the state-format version
  and the application ID, not on `app.version`.
- With the field absent or `false`, no state file is read or written and no
  window subclass is installed.
- A valid record restores the normal rectangle, maximized state, and DPI-scaled
  placement within the current work area.
- A missing, malformed, oversized, wrong-application, or wrong state-format
  version record is ignored and the manifest size is used.
- A minimized close restores a visible normal or previous-maximized window,
  never minimized.
- A cancelled close does not write, and a normal `WM_DESTROY` writes once.

Record exact commands, versions, and any skipped check. Do not present planned
evidence as completed.

## Rollback or Fallback

- Removing `rememberState` or setting it to `false` disables the feature and
  restores the current behavior; no migration is needed.
- Deleting `velox-window-state.json` resets the window to the manifest size.
- If restore logic proves unreliable, keep the field but ignore the record,
  falling back to the manifest size.

## Revisit Triggers

- A broader native capability, backend, or IPC method is proposed.
- Window state is requested for multi-display, per-monitor, or non-Windows
  targets.
- The state file needs fields, encryption, or application-visible access.
- The bounded 4096-byte load or one-shot write proves insufficient.
- A general persistent settings store is proposed.

## Synchronized Surfaces

- `docs/adr/README.md` (ADR table for 0017 and 0021).
- `docs/adr/0017-continue-as-a-narrow-static-packager.md` (narrow amendment).
- `docs/cli/configuration.md` (`window.rememberState` field).
- `docs/product/02-spec.md` (runtime window-state behavior).
- `schema/velox-v1.schema.json` (manifest `window.rememberState`).
- `schema/runtime-config-v1.schema.json` and
  `internal/runtimeconfig/config.go` (runtime-config parse and build
  passthrough).
- `internal/manifest/manifest.go` (manifest field and parse).
- The Windows host window-state code in `internal/webview2`.
