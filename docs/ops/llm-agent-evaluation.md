# Clean-Room LLM Agent Evaluation

- Status: Beta technical-evidence contract ready; no qualifying trial set recorded
- Owner: Project maintainer
- Decision: ADR 0018

## Purpose

Velox cannot schedule when an unrelated person will volunteer to test an alpha.
The beta technical-readiness gate therefore uses fresh coding-agent sessions to
test whether a capable LLM can discover, author, package, inspect, and run a
non-trivial application from the public release alone.

This is agent-usability evidence, not human adoption. It does not prove demand,
trust, documentation quality for people, willingness to tolerate SmartScreen,
or commercial viability.

## Evaluation Unit

One trial uses:

- one fresh LLM session with no conversation or memory carryover;
- one fresh Windows workspace, VM, or hosted runner;
- one immutable public Velox release URL and independently recorded ZIP
  SHA-256;
- the public `docs/QUICKSTART.md` discovery entrypoint;
- the public task at `evals/llm-agent/v1/task.md`;
- no Velox source checkout, unpublished context, local release output, or
  interactive maintainer hint;
- one schema-valid `velox.llm-agent-evaluation/v1` result.

Every trial receives one immutable series ID and a unique sequence from 1
through 3. Failed and held sequences remain part of the series and cannot be
overwritten by a later pass.

The orchestrator may supply credentials needed by the LLM provider, but those
credentials never enter the trial workspace, result, artifact, or repository.
The trial itself must not receive repository write, release, signing, or secret
authority.

## Verdicts

- `passed`: every schema hard gate is true, required artifacts exist, the two
  build hashes match, application behavior is observed, and no forbidden action
  occurred.
- `failed`: a reproducible product, documentation, compatibility, safety, or
  task failure occurred.
- `held`: the environment or evidence was insufficient to decide. A held trial
  is preserved but does not count as pass or fail.

An LLM's final prose, self-review, confidence, or claim of completion is never
an oracle. Deterministic hashes, CLI results, final files, process outcome, and
observable application state own the verdict.

The repository-owned verifier in `scripts/verify-llm-agent-evaluation.ts`
recomputes prompt and artifact hashes, rejects unsafe paths and symbolic links,
checks pass-gate consistency, and derives the three-trial series verdict. A
target-specific Mustflow intent must bind real result paths before the verifier
is run against trial evidence.

## Beta Gate

The beta technical gate passes only when all of the following are true:

1. Three consecutive trials pass against the same release bytes, task version,
   and result schema.
2. At least two distinct model identifiers are represented. Distinct prompts
   or sessions of one model do not count as model diversity.
3. Every trial starts fresh and records no memory carryover, source checkout,
   unpublished context, or maintainer intervention.
4. Every trial verifies checksum, public-doc discovery, no consumer toolchain,
   deterministic build, inspection, startup, and Focus Ledger behavior.
5. No trial records a forbidden action, hidden native capability, sensitive
   evidence, or unclassified failure.

Do not discard a failed trial and keep sampling until three convenient passes
appear. A product, prompt, release, schema, or documentation change starts a
new consecutive-trial series. Held infrastructure trials remain visible and
may be replaced only after their hold reason is recorded.

## Stable Gate

Stable consideration requires the beta gate to pass on at least two immutable
public releases with no unresolved critical product or security risk. A human
attempt is welcome market evidence but is not a calendar-dependent technical
gate under ADR 0018.

## Evidence Packet

Store only:

- the public task version and SHA-256;
- the immutable series ID and sequence;
- provider and model identifiers;
- a hash of the session identifier, never the raw session token;
- public release identity and observed digest;
- redacted environment versions;
- hard-gate booleans, command classes, counts, stable diagnostics, relative
  artifact paths, and artifact hashes;
- a concise report without chain of thought, full transcript, private path, or
  secret value.

Raw prompts containing provider credentials, complete tool payloads, full
transcripts, screenshots with personal data, and local absolute paths are not
evaluation artifacts.

## Failure Handling

- Checksum mismatch fails before execution.
- Missing public instructions fails `publicDocsOnly`; the maintainer may fix the
  docs and start a new series.
- Unsupported WebView2 or hosted-environment failure is `held` only when the
  environment evidence proves the product path was not reached.
- A compiler, Node.js, package manager, source checkout, or hidden maintainer
  hint fails the trial even if the final application works.
- An unsafe or unverifiable trajectory cannot be repaired by an LLM judge's
  favorable explanation.

## Human Evidence

`docs/ops/external-user-attempt.md` remains available for voluntary reports.
Such reports may change product positioning, signing priority, support policy,
or the decision to continue, but their absence no longer blocks beta technical
readiness.
