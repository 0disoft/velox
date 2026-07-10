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
- Host candidate: pure Go with no CGo.
- Host fallback: C++23 after an explicit M0 gate.
- Web runtime: installed Evergreen WebView2.
- Frontend bridge: dependency-free JavaScript.
- IPC: bounded JSON request-response over direct WebView2 messages.
- Packaging: portable directory and deterministic ZIP.

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

This document describes intended architecture. It does not claim that the host,
CLI, deterministic build, security controls, or performance targets have been
implemented. M0 evidence determines whether the design proceeds.
