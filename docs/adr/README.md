# Architecture Decisions

- Status: Active
- Owner: Project maintainer

## Purpose

Architecture decision records own durable choices that constrain product,
implementation, compatibility, security, performance, or release behavior.

Product scope remains owned by docs/product/02-spec.md. An ADR explains a
choice inside that scope and cannot silently add a product capability.

## Decisions

| ADR | Status | Decision |
| --- | --- | --- |
| 0001 | Superseded by 0005 | Windows x64, Go CLI, experiment-gated pure-Go host, unchanged generic host, WebView2, no local server |
| 0002 | Accepted | One primary source for each product, architecture, CLI, performance, validation, and scaffold contract |
| 0003 | Accepted for M0 | Pin go-webview2 for startup feasibility only; do not treat the wrapper as the production security boundary |
| 0004 | Accepted for M0 | Keep a direct C++23 WebView2 host as the security, lifecycle, and startup reference |
| 0005 | Accepted | Use Go for both the CLI and production host; keep the WebView2 adapter repository-owned |

## Lifecycle

- Proposed: under review and not binding.
- Accepted: current binding direction.
- Accepted for M0: binding only for the named experiment or milestone.
- Superseded: replaced by a newer ADR.
- Rejected: considered and intentionally not selected.
- Retired: no longer relevant to active architecture.

Accepted ADRs are not proof of implementation.

## Required Content

- Context and decision pressure.
- Explicit decision and scope.
- Rejected alternatives.
- Positive and negative consequences.
- Validation or evidence gate.
- Revisit and supersession triggers.
- Owner and decision date.

## Review Blockers

- The decision contradicts product scope without changing the product source.
- A performance or compatibility claim has no evidence gate.
- Migration, fallback, or removal path is missing for an experiment.
- Security or native capability impact is omitted.
- Derived docs and diagrams are not synchronized.

## Template

Use docs/adr/0000-template.md and assign the next numeric identifier. Do not
reuse identifiers or edit an accepted decision to hide a reversal; supersede it.
