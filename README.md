# Velox

- Status: Go runtime feasibility proven; packaging MVP not implemented
- Scope: general
- Repository type: cli-tool

Velox is a working name for a compile-free Windows desktop application
packager. It is designed to turn static HTML, CSS, and JavaScript into a
portable WebView2 application without compiling application-specific native
code.

The repository now contains a policy-enforcing pure-Go WebView2 host, a direct
C++23 benchmark reference, a strict external runtime configuration parser, a
dependency-free hello fixture, and a named-pipe startup benchmark harness. It
is not an alpha distribution; packaging commands and the broader IPC contract
are still design work.

## Priorities

1. End-to-end cold build time.
2. Consumer GitHub Actions cache upload.
3. Process-to-ready application startup.

Velox intentionally trades native feature breadth for a smaller build and
runtime surface. Startup is a hypothesis to measure, not a proven advantage.

## Proposed Shape

- A standalone Go CLI validates and packages projects.
- A separate prebuilt generic host opens static assets through WebView2.
- The production host is pure Go with no CGo or C++ shim.
- A minimal C++23 host remains an M0 benchmark reference, not a product
  fallback selected by startup numbers alone.
- Consumer builds copy an unchanged host, external configuration, and assets.
- Consumer builds require no compiler, Node.js, or frontend package manager.

## Current Product Boundary

Supported by the MVP design:

- Windows x64.
- Static web assets.
- One top-level window.
- Portable directory and deterministic ZIP output.
- Minimal versioned JSON IPC for application information and basic window
  lifecycle.
- Non-interactive CLI operation and machine-readable output.

Explicitly deferred:

- Native application backends and plugins.
- Filesystem, shell, process, and sidecar APIs.
- Frontend bundling, hot reload, and a development server.
- Installers, automatic updates, per-application executable branding, and code
  signing automation.
- macOS, Linux, ARM64, and multi-window support.

## Documentation

- Product scope: docs/product/02-spec.md
- Product brief: docs/product/00-product-brief.md
- Roadmap: docs/product/01-roadmap.md
- Risk register: docs/product/03-risk-register.md
- Architecture: ARCHITECTURE.md
- Initial decision: docs/adr/0001-initial-architecture-boundaries.md
- CLI contract: docs/cli/command-contract.md
- Performance budget: docs/engineering/03-performance-budget.md

## Development State

M0 is a feasibility and kill test. It must compare a pure-Go WebView2 host with
a minimal C++23 reference host and determine whether Velox has a meaningful
advantage over Wails, existing compile-free wrappers, and a PWA.

No consumer installation, packaging, or release command is documented yet.
The parent workspace exposes bounded maintainer-only build, smoke, and
benchmark intents documented in `DEVELOPMENT.md` and `VALIDATION.md`.

## Repository Workflow

- Agent instructions: AGENTS.md
- Validation names: VALIDATION.md
- Checklist router: CHECKLIST.md
- Documentation index: docs/README.md
- Scaffold state: .ssealed/manifest.json
