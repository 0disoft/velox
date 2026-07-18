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
| 0004 | Retired by 0005 | Bounded C++23 WebView2 reference used for the completed M0 comparison |
| 0005 | Accepted | Use Go for both the CLI and production host; keep the WebView2 adapter repository-owned |
| 0006 | Accepted | Bind the CLI and host with strict release metadata |
| 0007 | Accepted | Retain virtual HTTPS asset delivery while immediate-relaunch recovery is diagnosed |
| 0008 | Accepted | Pass structural simplicity only for the portable static-app topology and keep the PWA counterargument explicit |
| 0009 | Accepted | Remove startup from the product headline and retain it as a release guardrail |
| 0010 | Accepted for future signed channels; M4 gate superseded by 0011 | Separate GitHub build attestation from SignPath Authenticode signing and preserve both release lineages |
| 0011 | Accepted | Publish an explicitly unsigned developer preview before provider onboarding or signing |
| 0012 | Accepted for evidence rules; candidate superseded by 0014 | Bind preview tags to embedded versions and separate public-download evidence from an external-user attempt |
| 0013 | Superseded by 0014 | Block a public executable under the colliding Actutum working name |
| 0014 | Accepted | Adopt Actutum as the public product, command, module, schema, and release identity |

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
