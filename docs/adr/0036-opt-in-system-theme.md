# ADR 0036: Opt-In Native System Theme

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Decision

Add optional window.followSystemTheme boolean to the manifest and packaged
runtime config, default false, preserving contract version 1. Omitted or false
loads no theme API or window subclass and keeps the current default title bar.
True requests a native title bar that follows the current Windows app light/dark
setting through the documented DWM immersive-dark-mode attribute
(DWMWA_USE_IMMERSIVE_DARK_MODE, attribute 20). That attribute is documented for
Windows 11 build 22000 and newer; on older or unsupported builds the host
retains the default title bar instead of failing. This is a title bar
appearance preference, not a security boundary and not a promise of an exact
color value. All application content theming stays app-owned.

## Host Behavior

With true, the host listens for WM_SETTINGCHANGE, WM_THEMECHANGED and
WM_SYSCOLORCHANGE on the existing UI thread. It adds no polling loop, timer,
worker, sidecar or dependency. On each event it reads the AppsUseLightTheme
value read-only from the current user's personalization key and queries
SPI_GETHIGHCONTRAST through SystemParametersInfo. A missing preference reads as
light. High contrast disables the dark title bar override so the system scheme
wins. A read error keeps the last successful appearance instead of forcing a
change. Native attribute writes are deduplicated, so repeated or unrelated
events do not re-request the same appearance. The host may reapply the
attribute after the OS resets non-client styles, including a same-theme
WM_THEMECHANGED.

Scope stays narrow:

- No public native IPC method or permission; standard browser media queries remain available.
- No database, persistent state or state-format change.
- No external font, network fetch or bundled theme asset.
- No injected CSS, application-style mutation or forced color adjustment.

Application CSS remains app-owned. An app can style light and dark with
prefers-color-scheme and handle high contrast with forced-colors; the host does
not suppress that with forced-color-adjust.

## Alternatives

### Force the application document dark or light from the host

Rejected. It would override app-owned CSS and expand the runtime boundary
beyond the title bar.

### Poll the personalization key on a timer

Rejected. The change messages already exist on the UI thread; polling adds
background work with no new information.

### Always request the dark attribute regardless of build support

Rejected. The attribute is not documented before Windows 11 build 22000, so an
unsupported build must keep the default title bar.

### Add a public IPC method or permission

Rejected. This is a host appearance preference with no script-controlled
behavior to expose.

## Consequences

### Positive

- One opt-in boolean gives a native-looking title bar that follows the Windows
  app theme on supported systems.
- Default behavior, bundled config bytes and host startup for existing apps are
  unchanged.
- The host reuses the existing UI thread and change messages, so there is no new
  process, timer or dependency.

### Negative

- Exact title bar rendering depends on the OS build and theme; the host can only
  request the appearance and cannot guarantee pixels.
- Unsupported or older Windows builds keep the default title bar even when the
  field is true.
- One more opt-in window option to validate and document.

## Validation

Cover omission/false/true serialization and default propagation, invalid types,
older or unsupported-build fallback, AppsUseLightTheme light/dark reading, a
missing preference reading as light, high-contrast priority, error retention of
the last successful appearance and deduplicated native writes. Confirm no script
method or permission is added and that no persistent state or dependency
appears. Live user-driven Windows theme changes and visual title bar comparison
remain manual evidence.

## Rollback or Fallback

Remove window.followSystemTheme to return to the default title bar. No profile
migration or state cleanup is required, because the option stores nothing. The
fallback is the retained default title bar on unsupported builds.

## Revisit Triggers

- Microsoft changes or removes the documented dark title bar attribute or its
  supported build floor.
- A supported, documented API appears for full non-client theme following.
- Users need script-controlled or per-window theme selection.

## Synchronized Surfaces

- docs/cli/configuration.md and docs/product/02-spec.md window fields.
- VALIDATION.md standard validation names.
- README.md and docs/adr/README.md indexes.
- examples/window-system-theme.
