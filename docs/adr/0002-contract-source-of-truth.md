# ADR-0002: Contract Sources of Truth

- Status: Accepted
- Date: 2026-07-10
- Owner: Project maintainer

## Context

Velox has intentionally duplicated summaries for readers, but product scope,
CLI behavior, protocol details, and performance thresholds cannot have multiple
equal authorities. Drift would let implementation or marketing choose the most
convenient statement.

## Decision

Each durable contract has one primary source.

| Contract | Primary source | Derived surfaces |
| --- | --- | --- |
| Product scope and non-goals | docs/product/02-spec.md | README.md, product brief, roadmap |
| Architecture decisions | docs/adr/ | ARCHITECTURE.md, architecture docs, diagrams |
| Cross-cutting invariants | docs/engineering/00-project-invariants.md | checklists, tests, review docs |
| CLI commands and options | docs/cli/command-contract.md | CLI README, help, examples, tests |
| CLI configuration | docs/cli/configuration.md | manifest schema, help, fixtures |
| CLI output and exit codes | docs/cli/output-and-exit-codes.md | command implementation, tests, examples |
| Performance definitions and gates | docs/engineering/03-performance-budget.md | README claims, benchmark summaries |
| Validation names | VALIDATION.md | checklists, CI, contribution docs |
| Scaffold ownership | .ssealed/manifest.json | ssealed doctor and update output |

Until implementation exists, prose contracts are authoritative for intended
behavior only. Once machine-readable schemas and executable command behavior
exist, they become authoritative for syntax and runtime behavior; prose must be
synchronized to them.

## Rules

- Summary documents link to the primary source and do not restate long
  procedures.
- A derived surface may simplify wording but cannot add capability.
- Unsupported behavior remains unsupported when a summary omits the limitation.
- Numeric claims cite the benchmark revision that produced them.
- Generated help, schemas, fixtures, and examples change in the same review as
  their owning contract.
- When sources conflict, release is blocked until the primary source and
  implementation agree.

## Consequences

- Some information is intentionally repeated for discoverability.
- Reviews must identify both the changed primary contract and synchronized
  derived surfaces.
- Machine-readable contracts will gradually replace prose as syntax authority.
- The repository may remove redundant documents rather than maintain parallel
  contracts.

## Validation

Documentation review checks exact command names, option names, exit codes,
version fields, metric names, thresholds, and support boundaries across primary
and derived surfaces.
