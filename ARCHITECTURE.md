# Architecture

- Status: Draft
- Owner: Project maintainer

## Summary

Velox separates build-time packaging from runtime hosting.

The build side is a standalone Go CLI. The runtime side is a distinct prebuilt
generic host. They share versioned contracts but do not share executable
responsibilities.

## Build Boundary

The CLI owns:

- Project manifest parsing and validation.
- Asset and path validation.
- Host compatibility selection.
- Immutable build planning.
- Atomic output staging.
- Static asset copying.
- Build reports and deterministic archives.

It does not compile application code, run a frontend package manager, download
dependencies during a build, or modify source assets.

## Runtime Boundary

The host owns:

- Windows window lifecycle.
- WebView2 lifecycle and security settings.
- Virtual-host mapping for local assets.
- Trusted-origin message validation.
- A small, closed native method table.

It does not contain CLI packaging code, parse the authoring manifest, run a
local HTTP server, or expose arbitrary native capabilities.

## Initial Stack

- Platform: Windows x64.
- CLI: Go.
- Production host: pure Go with no CGo or C++ shim.
- WebView2 binding: repository-owned adapter over a pinned, maintained pure-Go
  fork.
- Web runtime: installed Evergreen WebView2.
- Frontend bridge: dependency-free JavaScript.
- IPC: bounded JSON request-response over direct WebView2 messages.
- Packaging: portable directory and deterministic ZIP.

`internal/webview2` owns the host-facing adapter contract. The executable may
depend on that package but must not import the checked-in WebView2 fork
directly. The fork lives under `third_party/go-webview2`, is pinned through a
Go module replacement, and exposes only the low-level behavior needed by the
adapter. Security policy remains in `internal/webview2`, not in CLI or runtime
configuration layers.

The M1 build path is split across `internal/manifest`, `internal/assettree`,
`internal/buildplan`, `internal/builder`, and `internal/archive`. Parsing and
filesystem discovery complete before the builder mutates the output root. The
builder copies the release-bundled host without patching it, emits a separate
runtime configuration, assembles an owned sibling staging directory, writes a
normalized ZIP, and then promotes the directory and archive while preserving
recoverable previous outputs on handled failures.

The release bundle binds the CLI and host through `velox-host.json`. The CLI
rejects a host when release version, target, host contract, runtime contract,
file size, or SHA-256 differs. The sidecar is not a signature and does not make
an untrusted download trustworthy. The unsigned developer preview ships an
external checksum, SBOM, and provenance metadata with an explicit warning; a
later signed channel adds publisher identity without changing the internal
compatibility contract.

The retired C++23 M0 comparison remains documented in ADR 0004 and the
performance budget. Its source and Pixi toolchain are no longer active
repository dependencies.

## Contract Sources

- Product scope: docs/product/02-spec.md
- System boundary: docs/architecture/00-system-boundary.md
- Domain model: docs/architecture/01-domain-model.md
- Runtime flow: docs/architecture/02-runtime-flow.md
- Quality attributes: docs/architecture/03-quality-attributes.md
- Architecture decisions: docs/adr/
- Project invariants: docs/engineering/00-project-invariants.md
- Performance budget: docs/engineering/03-performance-budget.md

## Diagrams

- System context: diagrams/system-context.mmd
- Container view: diagrams/container-view.mmd
- Core runtime flow: diagrams/core-runtime-flow.mmd

## Evidence Boundary

This document distinguishes implemented slices from intended architecture. The
Go host security boundary, seven-command M1 CLI, host release metadata,
deterministic consumer bundle, and end-to-end evidence contracts are
implemented and covered by local tests and the hosted consumer workflow.
Public cross-framework benchmark fixtures are complete. Public unsigned
distribution and independent external-user evidence remain incomplete; signed
distribution is deferred by ADR 0011.
