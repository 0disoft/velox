# ADR 0019: Gate beta on product workflows

- Status: Accepted
- Date: 2026-09-10
- Owner: Project maintainer
- Supersedes in part: ADR 0018 beta and stable channel admission requirements

## Context

Velox is a narrow static Windows desktop packager, not an agent evaluation
platform. Session adapters, model routing and evaluator isolation grew beyond
the product work they were intended to assess. A model-specific trial series
is neither necessary nor sufficient to demonstrate a usable desktop app.

## Decision

Beta readiness is owned by the product workflow checklist in
`docs/ops/product-readiness.md`. AI evaluation is optional diagnostic evidence,
not a release prerequisite. Preserve historical runs, failures, v1/v2 schemas
and isolation checks; do not relabel their evidence or relax their validators.
Legacy `betaTechnicalGate` fields describe the ADR 0018 evaluation contract
only and cannot authorize current channel promotion.

Require source-free public-release creation, development, reproducible build,
inspection and launch; File Notes open/edit/save/cancel/denial/recovery checks;
and bounded Windows shutdown/relaunch checks with no unresolved critical
security or data-loss issue. Record exact versions, artifacts, results and
limitations. Mocked file APIs do not prove native picker behavior.

No automatic beta or stable promotion follows from this decision. Stable
requires the product checks on at least two immutable public releases, no
unresolved critical risk, and a separate maintainer support/signing decision.
Unsigned-alpha warnings, native permission boundaries, origin restrictions
and the static-only product scope remain unchanged. Human adoption is still
unproven; maintainer testing must not be described as independent adoption.

## Alternatives

- Keep model attestation mandatory: rejected because provider plumbing blocks
  product iteration without proving native application behavior.
- Remove all gates: rejected because saved-data integrity, shutdown and public
  distribution failures directly affect users.

## Consequences and Validation

Development now prioritizes user-visible workflows. Maintainer bias remains;
preserve failures and limitations rather than claiming external validation.
The readiness checklist must distinguish local source tests, native startup,
real user-gesture file operations and public-release evidence. Existing tests
remain available; evaluation tools are paused, not deleted or silently trusted.

Revisit this decision if independent usage identifies a missing product gate,
or if agent evaluation becomes a separately justified product requirement.
