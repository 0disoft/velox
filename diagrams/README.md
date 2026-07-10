# Diagrams

- Status: Draft
- Owner: Project maintainer

## Role

Diagrams are derived summaries. Product and architecture prose remain
authoritative when a diagram conflicts with a source document.

## Active Diagrams

- system-context.mmd: actors, Velox boundary, output, Windows runtime, and
  application-owned network services.
- container-view.mmd: CLI, release bundle, output, host, bridge, WebView2, and
  Windows boundaries.
- core-runtime-flow.mmd: host startup, trust checks, ready state, IPC, failure,
  and shutdown.

## Retired Diagrams

- release-flow.mmd: retired until release automation exists.
- rollback-flow.mmd: retired until promotion and rollback tooling exists.

## Sources

- Product scope: docs/product/02-spec.md
- System boundary: docs/architecture/00-system-boundary.md
- Runtime flow: docs/architecture/02-runtime-flow.md
- Architecture decisions: docs/adr/
- Operations: docs/ops/

## Rules

- Do not add implemented-state details before implementation exists.
- Keep component and command names exact.
- Keep trust boundaries and failure branches visible.
- Do not use diagrams as benchmark evidence.
- Update a diagram in the same change as its owning architecture contract.
