# Product Specification

- Status: Draft
- Owner: Project maintainer
- Public name: Velox

## Product Statement

Velox packages static HTML, CSS, and JavaScript as a Windows desktop
application without compiling user-owned native code during the application
build.

The first product is a narrow CLI and runtime pair:

- A prebuilt Go CLI validates and packages a project.
- A prebuilt generic pure-Go host opens the project through WebView2.
- An external manifest and asset directory remain separate from the unchanged
  host executable.

## User Problem

A small desktop application can inherit a disproportionately large build
surface from its wrapper: language toolchains, package installs, generated
bindings, caches, platform SDKs, and intermediate output. On clean CI runners,
that surface costs time and storage before the application's own assets are
processed.

Velox removes application-specific native compilation from the normal consumer
build path. It accepts a smaller feature set in exchange.

## Target Use Cases

- Offline documentation and media viewers.
- IndexedDB-backed local-first tools.
- Agent-authored small applications whose behavior remains inspectable as
  static assets.
- Static dashboards and internal utilities.
- Portable prototypes and kiosk-style single-window applications.
- Applications whose native needs are limited to lifecycle and basic window
  control.

## Unsupported Use Cases

- Applications requiring an arbitrary Go, Rust, C++, or Zig backend.
- Background daemons, sidecars, or native plugin ecosystems.
- Heavy filesystem processing or shell and process execution.
- Applications requiring a bundled and version-pinned browser engine.
- Applications requiring macOS or Linux support in the first proof.
- Applications requiring tamper-resistant embedded web assets in the first
  release.

## MVP Capabilities

### Project input

- A versioned JSON manifest.
- A static asset root.
- One HTML entry point inside that root.
- Basic application identity and initial window settings.
- An explicit, closed set of native permissions.
- Optional per-application executable branding: an icon and a version
  resource.

### Build output

- A prebuilt host executable, copied unchanged by default or branded from a
  staged copy when `branding` is configured.
- An external runtime configuration file.
- A copied static asset directory.
- A machine-readable build report.
- A deterministic portable ZIP archive.
- An optional per-user Windows Setup executable (`<app-id>-setup.exe`) when
  `velox build --installer` is passed.

The M1 implementation produces these outputs for `windows-x64`. An unsigned,
deterministic consumer release bundle now carries strict host metadata and the
CLI verifies its release, target, contract versions, size, and SHA-256 before a
build. The alpha-evidence path now emits checksums, SPDX, and an unsigned
provenance statement. The first public artifact is the explicitly unsigned
`0.5.10-alpha.1` developer preview. Its immutable tag, public publication, and
same-repository no-checkout verification are complete. A separate public,
now-archived maintainer-controlled clean-room consumer run also completed the
documented public-release path without Velox source or a consumer toolchain.
ADR 0016
therefore closes the technical M4 distribution milestone while recording zero
independent-user attempts. Signatures and authenticated provenance are future-
channel work; sidecar metadata alone is not a trust anchor.

ADR 0017 continues alpha development as a narrow static desktop packager. It
does not approve an application-specific Go backend or a broader native
capability surface. ADR 0019 makes product workflows the beta technical gate;
AI evaluation is optional and does not approve a channel. The checklist is
`docs/ops/product-readiness.md`. Passing checks does not prove market demand.
The retained historical evaluation path uses a maintainer-built Windows AppContainer
supervisor and no-breakaway Job Object. This maintainer tool is not shipped in
the consumer release bundle and does not expand the application runtime API.

ADR 0020 amends ADR 0017 for two narrow surfaces: optional compiler-free
executable branding of a staged host copy on the Windows build, and an optional
per-user Windows Setup executable. The default portable output keeps the shared
Velox icon and no application-specific version metadata, and stays the default
build output; the Setup is an additional, opt-in artifact. Neither adds an
application runtime capability: the static-only asset model, the closed IPC v1
method table, and the no-consumer-compiler boundary are unchanged. Install,
removal, the isolated Setup payload, and the remaining limitations are
documented in `docs/ops/windows-installer.md`.

ADR 0021 adds a narrow, opt-in window-state persistence amendment to ADR 0017.
It restores one top-level window's placement from a bounded host-owned state
file and does not widen the application runtime API.

ADR 0022 adds a narrow, opt-in single-instance host amendment to ADR 0017. A
duplicate launch activates the existing window and exits without a second
WebView, and it does not widen the application runtime API.

ADR 0023 permits one additional opt-in IPC method: confirmed HTTPS link opening
through the OS default handler. It does not permit generic shell execution,
process arguments, native file access, a backend, or unrestricted protocols.

ADR 0024 adds an opt-in host-owned system tray with a fixed open/hide/quit menu,
without a new web IPC method or arbitrary native menus.

The current public artifact is the explicitly unsigned `0.5.10-alpha.2`
developer preview from commit `9bbb6bfcc1393058cb80d72c79df601caa970f2f`.
Publication run `29895087658` and public-download verification run `29895490556`
passed with ZIP SHA-256
`abd07aab653db7d67adf822e6a944a6f85f54c9fb0752cce367724fb0ce62fb7`.

### Runtime

- Windows x64.
- Windows 10 version 1709 build 16299 or newer clients, or Windows Server 2016
  build 14393 or newer servers.
- An installed Evergreen WebView2 Runtime.
- Minimum WebView2 Runtime `92.0.902.49`, required for the download-denial
  interface used by the security baseline.
- One top-level window.
- Explicit development mode (`velox run --debug`) requests browser cache bypass
  for that WebView so ordinary reload reads edited assets. Production cache
  behavior and virtual-host mapping are unchanged; this internal protocol call
  opens no debugging socket and does not clear cookies, drafts or file handles.
- The host opts into per-monitor DPI awareness before creating its window:
  V2 where supported, with V1 fallback for Windows Server 2016. Existing
  incompatible DPI overrides fail startup instead of silently bitmap-scaling.
- Window dimensions use 96-DPI logical units. The initial outer window size
  and size limits scale for DPI; monitor DPI changes apply the Windows-suggested
  physical rectangle and refresh WebView bounds. Font sizes remain CSS-owned.
- Opt-in window-state persistence. With `window.rememberState: true`, the host
  saves the raw physical screen normal rectangle, monitor work area, DPI,
  maximized state, and a state-format version in `velox-window-state.json`
  under the application profile, and restores it after the window is created
  but before entry navigation. The restore scales for the current DPI, clamps
  into the current display work area, and keeps a minimum normal size of 320 by
  240 logical units. The state-format version is independent of `app.version`,
  so application updates keep the placement. A minimized close never restores
  minimized. The record is written once on a normal window destroy; a cancelled
  close does not write and forced termination is not guaranteed to save. When
  the field is absent or `false`, no state file is read or written and no
  window subclass is installed. This adds no IPC method, native permission, or
  background process.
- Opt-in single-instance launch. With `app.singleInstance: true`, the host
  acquires one per-identity Windows named mutex before opening WebView2, where
  the identity is the user SID, local session, `app.id`, and canonical profile
  path. A duplicate launch creates no additional WebView or window, asks the
  existing window to restore from minimized and request foreground, and exits
  `0`; a duplicate during startup is suppressed without activation. Foreground
  focus is best-effort. This is a convenience mechanism, not a security
  boundary, and it adds no sockets, named-pipe server, argument delivery,
  background process, or web IPC method. A lock or attach failure fails startup
  instead of silently duplicating.
- A virtual HTTPS origin mapped to the local asset directory.
- Opt-in system tray (`window.tray: true`, default `false`). The host reuses the
  current window icon, displays the application name as its tooltip, and handles
  open/hide/quit through the existing UI thread. Selecting the icon opens the
  window; Hide window is explicit, and X/Alt+F4/Quit retain normal close and
  `beforeunload` behavior. A cancelled close keeps the icon. Explorer restart
  triggers re-registration; failure reveals the window and disables tray hiding
  until a later successful registration. Normal destruction removes the icon.
  With the field absent or false, no tray icon, subclass, restart-message
  registration, polling, or native menu is added. There is no additional process,
  dependency, timer, notification API, or IPC permission. Hidden content keeps
  running; reduced resource use is not promised. Single-instance activation also
  reveals a tray-hidden window. The File Notes source opts in; older packaged
  outputs are not implicitly updated.
- Virtual HTTPS remains the only production asset transport while
  immediate-relaunch recovery is diagnosed under ADR 0007; file URL loading is
  a benchmark control only.
- Direct WebView2 web messaging with no listening socket.
- Basic application information and window lifecycle methods only.

### Initial Beta Support Scope

The Windows and WebView2 versions above are technical execution floors, not a
beta support promise. The initial beta support scope, agreed on 2026-09-23 and
revised on 2026-09-24, is a Windows 11 x64 desktop version still serviced by
Microsoft, with an installed, updating Evergreen WebView2 Runtime. Display
support is limited to one physical monitor; scaling-specific visual checks are
best-effort follow-ups rather than beta release gates. File Notes 0.1.1 was
observed at 125% on public alpha.62; the maintainer later reported sharp,
unclipped rendering at 100% and 150% on another laptop. That laptop's exact
Windows/WebView2 versions and copied ZIP hash were not recorded, so this is
manual visual evidence rather than full configuration or workflow verification.
Windows 10, Windows Server,
ARM64, multiple monitors (including mixed-DPI movement), and fixed or absent
WebView2 runtimes are outside this beta support scope even if a configuration
can technically launch. An initial beta would use a portable, unsigned ZIP;
warn about unsigned execution and the known same-profile relaunch delay. This
scope decision does not authorize beta publication or mark unverified scaling
as passed; `docs/ops/product-readiness.md` owns the remaining checks.

### CLI

- init
- validate
- doctor
- run
- build
- inspect
- version

All listed M1 commands are currently implemented.

The command contract is defined in docs/cli/command-contract.md.

### Window Close

After initialization, title-bar close and Alt+F4 request browser-owned closure.
WebView2 must be allowed to run `beforeunload` and obtain any required user
consent before the host destroys the window. Cancellation keeps the document
alive; unavailable draft storage must not be treated as successful persistence.
The host posts teardown only after `WindowCloseRequested`, outside the native
COM callback. Repeated teardown requests must remain idempotent.

Initialization failures and explicit runtime shutdown retain a separate forced
cleanup path, without document consent. An unresponsive page or failed script
request does not authorize an automatic discard timeout for ordinary user close.

## Security Contract

Web content is not trusted merely because it is local.

- Remote top-level navigation, popups, downloads, and browser permission
  requests are denied by default.
- File System Access read/write requests from the trusted app origin use
  WebView2's default activation checks and consent flow, not an automatic host
  grant. Restored file handles can trigger permission requests without a user
  gesture; the host must not convert these into persistent denials.
  Untrusted origins, failed kind/origin reads, and all other permission kinds
  remain denied. This does not add native file IPC.
- The host accepts messages only from the expected top-level application
  origin.
- Frames do not receive native capabilities.
- The native method table is closed and permission checked.
- Arbitrary filesystem, generic shell/process execution, and network proxy
  methods are absent. `external.open` permits a validated HTTPS URI,
  with manifest permission and a host-owned per-request confirmation. Work is
  bounded to one pending confirmation per host and dispatched outside WebView
  callbacks; no timer, background worker, or new dependency is added.
- ADR 0025 adds opt-in `file.openText` (`file.open`): read one host-selected
  local UTF-8 text file up to 2 MiB. No caller path, full-path result, durable
  grant, write, directory, network/device or reparse-point file access is added.
  Native selection is deferred outside WebView callbacks; cancellation reads
  nothing and document replacement invalidates pending work. File Notes 0.2.0
  uses this read-only API; Open does not connect a future write target.
- IPC payload size, nesting, and in-flight request counts are bounded.
- ADR 0028 adds opt-in `folder.read`: select one local directory using native
  UI and retain a revocable document-scoped token, never a caller path or
  persisted grant. Selection is read-only and excludes linked/remote/offline
  locations. It does not authorize file contents, recursive traversal or watching.
  `folder.list` identity-checks that directory and returns immediate names/kinds
  under 128-entry/32-KiB limits, with explicit truncation and exclusion counts.
  Adding separate `folder.readText` enables `folder.openText` for immediate
  UTF-8 files up to 2 MiB, opened relative to the verified directory handle.
  Existing `folder.read` apps gain no content access. Reject traversal, streams,
  directories, reparse/offline/encrypted children and multiply-linked files;
  retain revocation, no recursion, no path result and no write grant.
- ADR 0027 adds explicit connected Save as/Save under `file.save`: retain only
  one document-scoped target, with no exposed path or restart-persistent grant.
  Verify file identity, size, write time and content digest before reuse;
  reject external changes without overwriting. Revoke on navigation/shutdown
  and explicitly on New. Restored drafts need a new selection. Legacy
  `saveText` remains per-call selection. File Notes 0.2.0 uses connected saving,
  releases the target on New/successful Open and persists only draft fields.
- ADR 0026 adds independent opt-in `file.save` and `window.velox.saveText`:
  stage at most 2 MiB of UTF-8 text within unchanged 64 KiB message bounds,
  then save only to a native-dialog selection with overwrite confirmation.
  No caller path, durable grant or save-to-last-path is added. Cancellation
  writes nothing; replacement failures can retain sibling recovery files.
  File Notes 0.2.0 uses ADR 0027's connected helpers over this staging engine.
- Production mode disables development tools unless explicitly enabled by a
  development-only run path.

The M2 implementation exposes a frozen `window.velox.invoke()` bridge. IPC v1
uses a closed method table for application information and basic window
lifecycle, ADR 0023's confirmed HTTPS opener, ADR 0025's selected-text reader,
ADR 0026's selected-text saver and ADR 0027's document-scoped save target. It validates
permissions before dispatch and bounds payload size,
JSON nesting, request identifiers, duplicate identifiers, and concurrent
requests. The wire and method contract is defined in
`docs/architecture/04-ipc-v1.md`.

The initial directory-assets mode does not claim resistance to a local attacker
who can modify the installed asset directory.

## Build Contract

- Consumer builds do not invoke a native compiler.
- Consumer builds do not require Node.js or a frontend package manager.
- Once a pinned Velox release bundle is available, build is offline.
- Source assets are never modified.
- Output is assembled in a sibling staging directory and promoted only after
  validation succeeds.
- Handled failure leaves the previous successful output intact. A later build
  reconciles supported process-interrupted rename states from retained
  `.previous` outputs before mutation. This does not claim power-loss safety or
  atomic visibility across the directory and ZIP paths.
- Paths outside the project and output roots are rejected.
- Equivalent normalized inputs and the same Velox release produce identical
  unsigned archive bytes.

## Data Ownership

- Source assets and application configuration belong to the application
  author.
- Build output belongs to the application author.
- The Velox CLI persists no user profile or telemetry.
- The WebView2 user-data directory belongs to the packaged application.
- The default WebView2 profile is stable and application-scoped at
  `%LOCALAPPDATA%\Velox\profiles\<app-id>` on Windows. Velox does not delete it
  automatically. Benchmarks and controlled runs may override the location with
  `VELOX_DATA_DIR`.
- When `window.rememberState` is true, the host keeps one bounded window-state
  record (`velox-window-state.json`) in that profile directory. It is
  host-owned operational data and is not exposed to web content.

## Success Criteria

The MVP is successful when all of the following are demonstrated on a pinned
benchmark environment:

- A clean runner can produce a portable application without installing a
  compiler or Node.js.
- The consumer workflow uploads zero bytes to GitHub Actions cache.
- End-to-end cold build is materially faster than the equivalent Wails sample.
- Installation and runtime structure stay within ADR 0008's narrow advantage
  over the closest compile-free comparison.
- Fresh-profile, settled warm-profile, and immediate-relaunch process-to-ready
  results remain release guardrails without being presented as a product
  advantage.
- The security contract is covered by executable tests.

## Stop Conditions

Pause feature development and reassess the product if:

- The consumer build requires application-specific native compilation.
- A practical implementation requires a local server or broad native API.
- The cold-build advantage over Wails is less than 3x in the agreed headline
  fixture after benchmark noise is controlled.
- The product cannot preserve ADR 0008's portable-artifact distinction from a
  PWA or existing compile-free desktop wrapper.
- Startup reliability measurements stop at a blank window instead of usable
  content.

## Deferred Decisions

- Package-manager publication and namespace reservation strategy.
- Asset sealing, machine-wide or elevation-requiring installation, code signing,
  and automatic updates.
- macOS and Linux feasibility.
- Whether any native API beyond basic window control belongs in core.

## Source of Truth

This document owns product scope and non-goals. Architecture documents may
explain how the scope is implemented but must not silently expand it.
