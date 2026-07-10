# Product Specification

- Status: Draft
- Owner: Project maintainer
- Working name: Velox

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

### Build output

- An unchanged prebuilt host executable.
- An external runtime configuration file.
- A copied static asset directory.
- A machine-readable build report.
- A deterministic portable ZIP archive.

### Runtime

- Windows x64.
- An installed Evergreen WebView2 Runtime.
- One top-level window.
- A virtual HTTPS origin mapped to the local asset directory.
- Direct WebView2 web messaging with no listening socket.
- Basic application information and window lifecycle methods only.

### CLI

- init
- validate
- doctor
- run
- build
- inspect
- version

The command contract is defined in docs/cli/command-contract.md.

## Security Contract

Web content is not trusted merely because it is local.

- Remote top-level navigation, popups, downloads, and browser permission
  requests are denied by default.
- The host accepts messages only from the expected top-level application
  origin.
- Frames do not receive native capabilities.
- The native method table is closed and permission checked.
- Filesystem, shell, process, and arbitrary network proxy methods are absent.
- IPC payload size, nesting, and in-flight request counts are bounded.
- Production mode disables development tools unless explicitly enabled by a
  development-only run path.

The initial directory-assets mode does not claim resistance to a local attacker
who can modify the installed asset directory.

## Build Contract

- Consumer builds do not invoke a native compiler.
- Consumer builds do not require Node.js or a frontend package manager.
- Once a pinned Velox release bundle is available, build is offline.
- Source assets are never modified.
- Output is assembled in a sibling staging directory and promoted only after
  validation succeeds.
- Failure leaves the previous successful output intact and removes only the
  staging directory owned by the current run.
- Paths outside the project and output roots are rejected.
- Equivalent normalized inputs and the same Velox release produce identical
  unsigned archive bytes.

## Data Ownership

- Source assets and application configuration belong to the application
  author.
- Build output belongs to the application author.
- The Velox CLI persists no user profile or telemetry.
- The WebView2 user-data directory belongs to the packaged application.
- The exact default profile path and cleanup policy remain UNDECIDED until the
  runtime spike validates Windows behavior.

## Success Criteria

The MVP is successful when all of the following are demonstrated on a pinned
benchmark environment:

- A clean runner can produce a portable application without installing a
  compiler or Node.js.
- The consumer workflow uploads zero bytes to GitHub Actions cache.
- End-to-end cold build is materially faster than the equivalent Wails sample.
- Installation and runtime structure are demonstrably simpler than the closest
  compile-free comparison.
- Fresh-profile and warm-profile process-to-ready results are published without
  hiding setup, failures, or outliers.
- The security contract is covered by executable tests.

## Stop Conditions

Pause feature development and reassess the product if:

- The consumer build requires application-specific native compilation.
- A practical implementation requires a local server or broad native API.
- The cold-build advantage over Wails is less than 3x in the agreed headline
  fixture after benchmark noise is controlled.
- The product cannot explain a meaningful advantage over a PWA or existing
  compile-free desktop wrapper.
- Startup claims depend on measuring a blank window instead of usable content.

## Deferred Decisions

- Minimum supported Windows release.
- Minimum supported WebView2 Runtime version.
- Public product name and package namespaces.
- Asset sealing, installers, code signing, and automatic updates.
- macOS and Linux feasibility.
- Whether any native API beyond basic window control belongs in core.

## Source of Truth

This document owns product scope and non-goals. Architecture documents may
explain how the scope is implemented but must not silently expand it.
