# Documentation

- Status: Draft
- Owner: Project maintainer

## Authority Order

1. Product scope and non-goals: docs/product/02-spec.md
2. Accepted architecture decisions: docs/adr/
3. Project invariants: docs/engineering/00-project-invariants.md
4. CLI behavior: docs/cli/command-contract.md
5. Performance thresholds: docs/engineering/03-performance-budget.md
6. Validation names and reporting: VALIDATION.md

Summary pages must not silently broaden these sources.

## Product

- docs/product/00-product-brief.md: concise problem and product boundary.
- docs/product/01-roadmap.md: evidence-gated implementation sequence.
- docs/product/02-spec.md: canonical scope, capabilities, and stop conditions.
- docs/product/03-risk-register.md: current product and engineering risks.

## Architecture

- docs/architecture/00-system-boundary.md: ownership and trust boundaries.
- docs/architecture/01-domain-model.md: stable concepts and invariants.
- docs/architecture/02-runtime-flow.md: build and runtime sequences.
- docs/architecture/03-quality-attributes.md: ranked quality requirements.
- docs/architecture/04-ipc-v1.md: frozen JavaScript bridge, wire envelopes,
  limits, methods, permissions, and stable errors.
- docs/adr/: durable decisions and revisit triggers.
- diagrams/: derived visual summaries.

## CLI

- docs/cli/command-contract.md: canonical commands, options, JSON envelope, and
  exit codes.
- docs/cli/configuration.md: manifest and precedence contract.
- docs/cli/output-and-exit-codes.md: human and machine output rules.

## Engineering

- docs/engineering/00-project-invariants.md: rules that implementation cannot
  weaken without an ADR.
- docs/engineering/03-performance-budget.md: benchmark definitions and
  provisional gates.
- docs/engineering/04-security-baseline.md: trust boundaries and release
  blockers.
- docs/engineering/05-testing-standard.md: required evidence by layer.
- docs/engineering/06-dependency-and-change-policy.md: dependency admission and
  removal policy.
- docs/engineering/07-operability-and-failure-standard.md: diagnostics and
  failure behavior.

## Operations

The current repository has no service deployment, database, user secrets, or
production telemetry. Active operational documents cover CI readiness,
artifact release, and rollback of immutable releases only.

- docs/ops/00-operational-contract.md
- docs/ops/ci.md
- docs/ops/release.md
- docs/ops/signing.md
- docs/ops/rollback.md

Service-oriented backup, environment, secret, incident, and observability
templates are intentionally retired until a real support surface requires them.

## Workflow

- AGENTS.md: repository-local agent rules.
- CHECKLIST.md: checklist router.
- VALIDATION.md: stable validation names.
- .agents/context-map.md: agent routing.
- .ssealed/manifest.json: scaffold ownership and accepted-checksum state.
