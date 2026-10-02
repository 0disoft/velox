# CLI Configuration

- Status: Active
- Repository type: cli-tool
- Owner: Project maintainer

## Configuration Sources

Velox has one project manifest, velox.json by default.

Precedence is:

1. Explicit command-line options.
2. Project manifest values.
3. Documented built-in defaults.

Velox does not merge parent-directory manifests, user-global configuration, or
environment-derived application permissions.

## Manifest Shape

The machine-readable contract is `schema/velox-v1.schema.json`. Its top-level
shape is:

    {
      "schemaVersion": 1,
      "app": {
        "id": "com.example.hello",
        "name": "Hello",
        "version": "1.2.3",
        "singleInstance": false
      },
      "branding": {
        "icon": "app.ico",
        "company": "Rodisoft",
        "description": "Hello desktop app",
        "copyright": "(c) 2026 Rodisoft"
      },
      "assets": {
        "root": "web",
        "entry": "index.html"
      },
      "window": {
        "width": 960,
        "height": 640,
        "rememberState": false,
        "tray": false
      },
      "security": {
        "permissions": []
      }
    }

This example matches the implemented parser and JSON Schema contract.

## Field Ownership

### schemaVersion

Required integer identifying manifest syntax and semantics. Unsupported newer
required versions fail closed.

### app

Application identifier, display name, and version. `app.id` names the packaged
executable (`<app.id>.exe`) and the output directory; `app.name` sets the window
title; and all three values are recorded in the build report. When `branding`
is present, `app.name` and `app.version` additionally supply the executable
version resource.

`singleInstance` is an optional boolean, default `false`. When `true`, the
host acquires one Windows named mutex per application, profile, user, and
session before opening WebView2. A duplicate launch creates no second window;
it asks the existing window to reveal itself, restore from minimized and request foreground,
then exits `0`. A duplicate during startup is suppressed without activation,
and foreground focus is best-effort because Windows controls it. The mutex is a
convenience mechanism, not a security boundary. A lock or attach failure exits
`6` rather than silently launching a duplicate. See ADR 0022.

### branding

Optional. When present on a Windows build, the build always writes a
`VERSIONINFO` resource (product name from `app.name`, version from
`app.version`) into a staged copy of the prebuilt host. Omitting `branding`
entirely leaves the release host resources untouched.

- `icon`: project-relative `.ico` path. The file must stay inside the project
  root, be at most 2 MiB, and hold one to 32 PNG or BITMAPINFOHEADER images
  whose dimensions match the ICO directory entry. Omitting `icon` retains the
  host template's existing icon resources while the version resource is still
  written.
- `company`, `description`, and `copyright`: optional text written to the
  version resource. `description` defaults to `app.name`.
- Text values are limited to 256 UTF-16 code units and cannot contain control
  characters.
- `app.version` must have a numeric core of one to four components, each 0 to
  65535. A trailing `-` or `+` suffix is kept in the displayed version string.
- Branding is Windows-only and refuses a signed host template, because editing
  resources invalidates the signature.

Branding applies to `velox build` only. `velox run` launches the generic
release host directly, so the development preview does not show the branded
icon or version.

### assets

Project-relative asset directory and HTML entry point. Both must remain inside
the canonical project root after validation.

### window

Initial width and height, plus optional window-state persistence.

- `width`, `height`: initial outer size in 96-DPI logical units. Zero or
  omitted values resolve to 960 by 640. Widths below 320 and heights below 240
  are rejected.
- `rememberState`: optional boolean, default `false`. When `true`, the host
  saves the raw physical screen normal rectangle, the monitor work area, the
  DPI, the maximized state, and a state-format version (currently `1`) to
  `velox-window-state.json` in the application profile directory
  (`VELOX_DATA_DIR` when set, otherwise
  `%LOCALAPPDATA%\Velox\profiles\<app-id>`), and restores it after the window
  is created but before entry navigation. The restore scales for the current
  DPI, clamps into the current work area, and keeps a minimum normal size of
  320 by 240 logical units. The state-format version is independent of
  `app.version`, so application updates keep the placement. A missing,
  malformed, oversized, wrong-application, or wrong-state-format-version record
  is ignored. A minimized close never restores minimized. When `false` or
  omitted, the host performs no state file I/O and installs no window subclass.
  See ADR 0021.

- `tray`: optional boolean, default `false`. When `true`, the host adds one
  notification-area icon using the current window icon and application name.
  Selecting it opens the window; its native context menu offers Open window,
  Hide window, and Quit. Hiding is explicit: X, Alt+F4, and Quit still use the
  normal close path, including the page's `beforeunload` handling. Quit reveals
  the owner first so a confirmation is visible. A cancelled close keeps the icon.
  Explorer restart re-registers the icon; failure reveals the window and prevents
  further tray hiding until registration succeeds. The icon is removed on normal
  destruction. Forced termination can leave a stale shell icon until Windows
  cleans it up. Foreground focus is best-effort. No timer, polling, extra process,
  notification API, or public IPC method is added. Hidden WebView content still
  runs; hiding does not promise lower CPU or memory use. See ADR 0024.

Resizable state, position policy, and background color are not manifest fields
in v1.

### security

A closed permission list and production browser settings. Unknown permissions
are errors, not warnings.

`app.info` allows application information, `window.basic` allows basic window
operations, and `external.open` opts into confirmed HTTPS link opening in the
default Windows browser handler. The default list remains empty. Each external
link requires a host-owned confirmation; no file, shell command, custom
protocol, or unrestricted process launch is permitted. See
[IPC v1](../architecture/04-ipc-v1.md#external-https-links) for URL limits and
the queued-response contract.

`file.open` opts into `file.openText`: one host-selected local UTF-8 file, up
to 2 MiB, read-only. The method accepts no path and returns no full path or
reusable grant. It does not authorize writes or directory access. See
[selected text files](../architecture/04-ipc-v1.md#selected-local-text-files).

## Path Rules

- Relative paths resolve from the manifest's project root.
- Absolute source paths are rejected.
- Parent traversal is rejected.
- Links, junctions, and reparse points that escape ownership are rejected.
- Windows reserved names, alternate data streams, invalid trailing characters,
  and case collisions are rejected.
- Output paths cannot overlap source assets.

## Command-Line Overrides

The MVP allows operational overrides such as manifest path, target, and output
root. Identity, permissions, and security policy are not silently overridden by
environment variables.

## Environment Variables

Production configuration does not depend on environment variables.

Development or benchmark-only variables may select a ready-marker channel or
diagnostic verbosity. They must be explicitly named, documented, bounded, and
ignored by production builds.

## Defaults

Defaults must be:

- Stable within a manifest major version.
- Visible through validate or inspect output.
- Representable in normalized machine-readable output.
- Defined by the schema or one shared implementation source.

No default may grant a native capability.

## Validation

Configuration validation separates:

1. JSON syntax.
2. Schema shape.
3. Semantic constraints.
4. Filesystem and target checks.
5. Host and runtime compatibility.

Each failure returns a stable diagnostic code and project-relative location
when available.

## Deferred Configuration

- Plugin declarations.
- Sidecars and native backends.
- Automatic-update and signing settings.
- Frontend build commands.
- Multiple windows.
- Remote application URLs.
- macOS and Linux targets.

The optional per-user Windows installer is already available through
`velox build --installer`; it does not require a separate manifest section.
