# Velox

- Status: Design and feasibility stage
- Scope: general
- Repository type: cli-tool

Velox is a working name for a compile-free Windows desktop application
packager. It is designed to turn static HTML, CSS, and JavaScript into a
portable WebView2 application without compiling application-specific native
code.

The repository currently contains product and architecture contracts, not a
working implementation.

## Priorities

1. End-to-end cold build time.
2. Consumer GitHub Actions cache upload.
3. Process-to-ready application startup.

Velox intentionally trades native feature breadth for a smaller build and
runtime surface. Startup is a hypothesis to measure, not a proven advantage.

## Proposed Shape

- A standalone Go CLI validates and packages projects.
- A separate prebuilt generic host opens static assets through WebView2.
- The first host candidate is pure Go with no CGo or C++ shim.
- A minimal C++23 host is the benchmark and fallback reference.
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

No installation, build, or release command is documented yet because no
implementation or executable runner exists.

## Repository Workflow

- Agent instructions: AGENTS.md
- Validation names: VALIDATION.md
- Checklist router: CHECKLIST.md
- Documentation index: docs/README.md
- Scaffold state: .ssealed/manifest.json
