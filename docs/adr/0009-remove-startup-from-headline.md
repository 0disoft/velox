# ADR 0009: Remove startup from the product headline

- Status: Accepted
- Date: 2026-07-18
- Owner: Project maintainer

## Context

Startup began as the third product metric. Current evidence does not support a
cross-framework advantage:

- Go and the retired C++23 reference differed by about 104 ms at fresh p50 in a
  local, fixed-order comparison without a publishable environment contract;
- both hosts showed similar multi-second immediate same-profile relaunch tails;
- the production host and comparison wrappers share WebView2 initialization
  costs;
- no repeated, pinned cross-framework process-to-ready result exceeds the
  documented 10% noise boundary.

The lifecycle harness is still valuable. It catches readiness regressions,
preserves failures, separates fresh, settled warm, and immediate relaunch, and
records browser/UDF cleanup behavior. That value does not turn startup into a
competitive claim.

## Decision

Remove process-to-ready startup from the product headline and M3 advantage
gate. The two headline metrics are:

1. end-to-end consumer cold-build time;
2. consumer GitHub Actions cache upload.

Startup remains the first runtime guardrail. Releases continue to measure
usable content after DOMContentLoaded and two animation frames, retain p50 and
p95 plus failures, and investigate a 10% or greater p95 regression. Immediate
same-profile relaunch remains a lifecycle diagnostic, not a warm-start or
marketing number.

No README, product brief, or release note may describe Actutum as faster to start
than another wrapper without a new accepted ADR backed by repeated pinned
cross-framework evidence whose advantage exceeds both 10% and observed noise.

## Consequences

- The M3 startup gate is complete by removing the unsupported claim.
- Startup instrumentation, history, profile comparison, and recovery diagnosis
  stay maintained as reliability evidence.
- Work is not prioritized merely to shave small host-local milliseconds from a
  WebView2-dominated path.
- A severe startup regression can still block release even though startup is
  not a product differentiator.

## Alternatives

### Keep startup as an unproven third headline metric

Rejected. Repeating a hypothesis in the priority list invites marketing copy to
outrun the evidence.

### Remove startup measurement entirely

Rejected. WebView2 lifecycle, profile locks, browser cleanup, and ready-marker
correctness remain material reliability risks.

## Revisit Triggers

- Ten or more isolated, pinned, cross-framework samples use the same ready
  boundary, fixture, profile state, and failure-preserving contract.
- The measured p50 and p95 advantage both exceed 10% and the observed
  environment noise.
- A startup optimization preserves the origin, IPC, profile, and security
  contracts instead of hiding delay outside the measured boundary.

## Synchronized Surfaces

- `README.md`
- `VALIDATION.md`
- `docs/product/00-product-brief.md`
- `docs/product/01-roadmap.md`
- `docs/product/02-spec.md`
- `docs/product/03-risk-register.md`
- `docs/architecture/03-quality-attributes.md`
- `docs/engineering/03-performance-budget.md`
- `docs/adr/README.md`
