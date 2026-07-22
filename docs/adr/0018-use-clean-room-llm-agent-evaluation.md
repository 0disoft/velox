# ADR 0018: Use clean-room LLM agent evaluation for beta readiness

- Status: Accepted
- Date: 2026-07-22
- Owner: Project maintainer
- Supersedes in part: the beta and stable admission evidence clause in ADR 0017

## Context

ADR 0017 continued Velox as a narrow static desktop packager but made beta or
stable admission depend on an independent human attempt or a later explicit
acceptance of zero-adoption risk. The project cannot control when an unrelated
person will volunteer. A calendar-dependent actor therefore blocks technical
iteration without producing a more reproducible usability test.

Velox is also designed to make small desktop applications easy for coding
agents to author without learning a native toolchain. A clean-room coding agent
is consequently a legitimate early evaluator of the product's public discovery
and packaging path.

An LLM trial is not independent human adoption. The maintainer chooses the
release, prompt, models, and execution time. Model families may share training
data and failure modes. Treating model count as independent demand would repeat
the evidence-labeling mistake ADR 0016 removed from M4.

## Decision

Replace the uncontrollable human-attempt requirement in the beta and stable
technical gates with the repository-owned clean-room LLM agent evaluation
contract in `docs/ops/llm-agent-evaluation.md`.

A qualifying trial must:

- start in a fresh session and isolated Windows workspace;
- receive only an immutable public release URL, expected SHA-256, public
  documentation, and the versioned public task;
- avoid Velox source checkout, current conversation memory, unpublished
  context, local build output, and interactive maintainer hints;
- use no consumer compiler, Node.js runtime, package manager, frontend bundler,
  or application-specific native backend;
- produce deterministic build, inspection, startup, application-behavior, and
  bounded trajectory evidence;
- write a schema-valid result that remains
  `maintainer-orchestrated-clean-room-llm-agent` and fixes
  `humanAdoptionClaim` to `false`.

Beta requires three consecutive passing trials against the same release and
task version, representing at least two distinct model identifiers. Stable
consideration requires the beta gate on at least two immutable public releases
and no unresolved critical risk.

Voluntary human attempts remain valuable market evidence. Their absence no
longer blocks the technical channel gate, but it remains visible in R-014 and
must not be converted into a claim of demand, adoption, or human documentation
usability.

## Alternatives

### Wait indefinitely for an unrelated person

Rejected as a technical gate. It delegates schedule control to an unknown
actor and does not guarantee a controlled or replayable evaluation.

### Call one maintainer-run LLM session an external user

Rejected. The orchestrator remains the maintainer, and one stochastic success
is weak capability evidence. Every result must explicitly deny a human adoption
claim.

### Use an LLM judge to grade the final answer

Rejected. Self-report and judge preference cannot prove the final application,
artifact digest, deterministic build, safe trajectory, or absence of hidden
toolchains. Deterministic evidence and observable final state own hard gates.

### Require three runs of one model

Rejected as the normal gate. Repeated sessions reduce luck but do not expose a
model-specific blind spot. At least two model identifiers are required unless a
future ADR explicitly accepts single-model evidence.

### Remove every adoption signal

Rejected. Human attempts remain optional but important market evidence and can
still trigger signing, support, repositioning, or project-stop decisions.

## Consequences

### Positive

- The project controls when a technical beta evaluation can run.
- The task, input boundary, result shape, and hard gates are replayable.
- Agent usability is tested against public release bytes rather than local
  source or maintainer memory.
- Model prose cannot override deterministic failures or unsafe trajectories.

### Negative

- The maintainer still orchestrates the evaluation and can bias the prompt or
  model selection.
- Multiple models may have correlated training data and tool behavior.
- Passing agent trials do not prove that a person wants, trusts, or understands
  Velox.
- Provider access, model cost, Windows isolation, and GUI observation remain
  operational inputs outside the product artifact.

## Validation

- `schema/llm-agent-evaluation-v1.schema.json` fixes the evidence label, fresh
  session boundary, no-memory boundary, no-source boundary, hard gates, safe
  trajectory summary, and false human-adoption claim.
- `schema/llm-agent-evaluation-series-v1.schema.json` fixes three-trial identity,
  model diversity, outcome counts, beta verdict, and false human-adoption claim.
- `evals/llm-agent/v1/task.md` is the public task and contains no provider
  credential, private path, Velox source instruction, or hidden maintainer step.
- `docs/QUICKSTART.md` fixes the immutable-release, checksum, extraction, public
  CLI, startup, and failure path used by a clean-room consumer.
- Hygiene tests parse the schema and require ADR, roadmap, product, operations,
  risk, and README wording to stay synchronized.
- A real beta gate remains pending until three qualifying trial records exist.

## Rollback or Fallback

If clean-room setup cannot reliably prevent memory or source leakage, hold the
agent gate and do not claim beta readiness. Keep alpha development under ADR
0017 while tightening isolation.

If trials pass but real feedback later shows the product has no value, retain
the technical evidence and stop or reposition the product. Do not reinterpret
the old LLM results as human demand.

## Revisit Triggers

- Three qualifying trial records are completed.
- A trial exposes public-documentation, packaging, startup, or safety failure.
- Only one model family remains available.
- Provider or host behavior prevents fresh-session or workspace isolation.
- Beta, stable, signing, package-manager, or commercial distribution is
  proposed.
- Human feedback materially contradicts the LLM evaluation.

## Synchronized Surfaces

- `README.md`
- `PRIVACY.md`
- `VALIDATION.md`
- `docs/QUICKSTART.md`
- `docs/README.md`
- `docs/adr/README.md`
- `docs/ops/00-operational-contract.md`
- `docs/ops/external-user-attempt.md`
- `docs/ops/llm-agent-evaluation.md`
- `docs/product/01-roadmap.md`
- `docs/product/02-spec.md`
- `docs/product/03-risk-register.md`
- `evals/llm-agent/v1/task.md`
- `schema/llm-agent-evaluation-v1.schema.json`
- `schema/llm-agent-evaluation-series-v1.schema.json`
- hygiene tests
