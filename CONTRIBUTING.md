# Contributing

- Status: Draft
- Owner: Project maintainer

## Project Stage

Velox is in design and feasibility work. Contributions should strengthen the
M0 product hypothesis, architecture boundary, security contract, benchmark
fairness, or smallest executable proof.

Feature breadth is not a current goal.

## Before Changing Files

1. Read AGENTS.md, CHECKLIST.md, VALIDATION.md, and .agents/context-map.md.
2. Read docs/product/02-spec.md and the relevant ADR.
3. Identify the primary contract and every derived surface.
4. State which validation names can run and which are unavailable.
5. Keep unrelated generated scaffold files unchanged.

## Suitable Contributions

- Corrections to product or architecture contradictions.
- Smaller and safer Go WebView2 host experiments.
- Reproducible benchmark harness work.
- Path, configuration, IPC, and lifecycle tests.
- Documentation that records evidence or a real decision.
- Removal of unnecessary dependencies or support promises.

## Contributions to Avoid

- Plugins, sidecars, arbitrary native backends, or broad OS APIs.
- Frontend frameworks, bundlers, hot reload, or development servers.
- macOS or Linux implementation before the Windows go-or-kill gate.
- Marketing claims without published benchmark evidence.
- Generated bindings or dependencies added only for convenience.
- Placeholder implementations that report success.

## Change Requirements

- Public command changes update CLI contracts, help, examples, and tests.
- Manifest or IPC changes include compatibility classification.
- Native capability changes require an ADR, threat-model update, and negative
  tests.
- Performance-sensitive changes include measurement or an explicit evidence
  gap.
- Skipped checks include the reason and remaining risk.

## Commits and Releases

Unless explicitly stated otherwise, intentional contributions submitted for
inclusion are licensed under the project's `MIT OR Apache-2.0` terms. No CLA or
DCO sign-off bot is currently configured. Release automation and public
versioning remain gated by `docs/ops/release.md`.

## Security Reports

Do not open a public issue containing a working exploit, credential, private
path, or sensitive application data. Follow `SECURITY.md` and use GitHub private
vulnerability reporting.
