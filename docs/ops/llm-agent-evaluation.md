# Clean-Room LLM Agent Evaluation

- Status: V2 enforced sandbox path implemented; beta held pending a qualifying three-trial series
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
- one orchestrator-computed session-ID SHA-256 and tool-call budget;
- the public `docs/QUICKSTART.md` discovery entrypoint;
- the public task at `evals/llm-agent/v1/task.md`;
- no Velox source checkout, unpublished context, local release output, or
  interactive maintainer hint;
- one schema-valid `velox.llm-agent-evaluation/v1` result.
- one `velox.llm-agent-evaluation-attestation/v1` file generated from the
  orchestrator session log outside the agent-controlled trial workspace.

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

An LLM's final prose, self-review, confidence, result booleans, or claim of
completion is never an oracle. Deterministic hashes, CLI results, final files,
process outcome, observable application state, and the external orchestrator
attestation own the observed verdict. The Hermes attestation is not proof that
unrecorded filesystem or process access was impossible.

The repository-owned verifier in `scripts/verify-llm-agent-evaluation.ts`
recomputes prompt and artifact hashes, rejects unsafe paths and symbolic links,
requires an attestation outside the trial root, compares provider, model,
session digest, timestamps, tool counts, retries, budget, and forbidden actions,
checks pass-gate consistency, and derives the three-trial series verdict. A
target-specific Mustflow intent must bind real result and attestation paths
before the verifier is run against trial evidence.

The v1 tool-call budget is exactly 70. The verifier is invoked as:

```text
bun scripts/verify-llm-agent-evaluation.ts trial <trial-dir> <public-task> <attestation>
bun scripts/verify-llm-agent-evaluation.ts series <series-dir> <public-task> <attestation-dir> <series-dir>/summary.json
```

For series verification, each external attestation filename must match its
trial-directory basename with `.json` appended.

## Hermes Attestation Generation

The maintainer generates a Hermes attestation only after the fresh evaluation
session is closed. The generator reads Hermes `state.db` in read-only mode and
writes exactly one new attestation outside the agent-controlled trial root:

```text
bun scripts/hermes-evaluation-attestation.ts \
  --state-db <absolute-hermes-state.db> \
  --session-id <actual-hermes-session-id> \
  --trial-root <absolute-trial-directory> \
  --output <absolute-attestation-directory>/<trial-directory-name>.json \
  --trial-id <orchestrator-trial-id> \
  --series-id <orchestrator-series-id> \
  --sequence <1|2|3>
```

The output directory must already exist. The command refuses a relative path,
an unfinished or non-root session, an output inside the trial root, a filename
that does not match the trial directory, a Hermes schema it cannot classify,
a mismatch between persisted tool calls and the Hermes session counter, and an
existing output file.

Hermes currently has two observed completion representations. A session is
finished when every selected session row has `ended_at`, or when the final
active message is an assistant message with `finish_reason=stop`. A final user
message, pending tool call, missing finish reason, or any other ambiguous state
remains unfinished. Maintainers can inspect only these non-content completion
fields with the configured `velox_hermes_completion_live_diagnose` intent; the
diagnostic never prints prompts, message content, tool arguments, or a raw
session ID.

The generator derives provider, model, session-ID hash, timestamps, actual tool
count, retries after failed calls, and forbidden-action codes from the stored
session chain. It classifies explicit shell commands, source checkout, later
maintainer messages, workspace escapes, and known Hermes editor side effects.
In particular, Hermes file tools that write `.js`, `.ts`, `.go`, or `.rs` can
implicitly invoke a runtime, package manager, or compiler and therefore count
as forbidden even when the model did not type that subprocess command.

The generator preserves a redacted canonical projection of the selected
session and message rows and hashes that exact projection. Identifiers and
message/tool payloads appear only as SHA-256 values, and working directories
appear only as inside/outside/missing scope labels. This lets a reviewer
recompute the digest without storing raw prompts, tool arguments, tool results,
reasoning, raw session IDs, local paths, or the database.
The generator is maintainer orchestration and uses Bun outside the consumer
trial; Bun must never be exposed to or invoked by the evaluated agent.

This adapter alone is not operating-system process telemetry. Its v1 evidence
fixes `observationLevel` to `session-log-heuristic` and `sandboxEnforced` to
`false`. It can classify the
Hermes tool calls and known implicit editor behavior stored in the session
ledger, but a future Hermes helper that launches an unrecorded subprocess needs
a classifier update. V1 trials remain useful diagnostics but cannot pass the
beta technical gate. A qualifying trial combines that projection with an
independently emitted `velox.eval-sandbox-receipt/v1` in
`velox.llm-agent-evaluation-attestation/v2`.
Likewise, a root Hermes session proves that no parent transcript was resumed;
the orchestrator still owns the separate requirement to launch the evaluator
without provider-side or profile-side memory carryover.

## Series Orchestration

`scripts/llm-agent-orchestrator.ts` keeps agent-owned trial files separate from
maintainer-owned prompts, bindings, and attestations. The qualifying sequence is:

```text
bun scripts/llm-agent-orchestrator.ts prepare \
  --evaluation-root <absolute-external-evaluation-root> \
  --task-path <absolute-public-task-file> \
  --task-url <immutable-public-task-url> \
  --release-tag <release-tag> \
  --release-url <immutable-release-zip-url> \
  --release-sha256 <release-zip-sha256>

bun scripts/llm-agent-orchestrator.ts stage \
  --series-root <absolute-series-root> \
  --sequence <1|2|3>

go build -trimpath -o <tool-dir>/velox-eval-sandbox.exe ./cmd/velox-eval-sandbox

<tool-dir>/velox-eval-sandbox.exe \
  --trial-id <trial-id> --series-id <series-id> --sequence <1|2|3> \
  --trial-root <absolute-trial-root> \
  --tool-root <absolute-evaluator-install-root> \
  --pass-env PATH --pass-env <provider-credential-variable> \
  --prompt <absolute-staged-prompt> \
  --state-db-export <absolute-external-temporary-state.db> \
  --receipt <absolute-external-sandbox-receipt.json> \
  --timeout 45m -- <absolute-evaluator.exe> <new-session-arguments> <exact-prompt-argument>

bun scripts/llm-agent-orchestrator.ts attest-sandbox \
  --series-root <absolute-series-root> \
  --sequence <1|2|3> \
  --state-db <absolute-external-temporary-state.db> \
  --sandbox-receipt <absolute-external-sandbox-receipt.json>

bun scripts/llm-agent-orchestrator.ts verify \
  --series-root <absolute-series-root> \
  --task-path <absolute-public-task-file>
```

`prepare` creates three immutable identities and `stage` writes the exact
maintainer-owned prompt before a session ID exists. The supervisor launches a
new evaluator session in an ephemeral AppContainer, assigns its process tree to
a no-breakaway Job Object, grants only the trial and selected tool roots, and
constructs an isolated environment. The exact prompt must be one command
argument. For Hermes, use noninteractive single-query mode, pass the new session
ID into system context, ignore user rules and configuration, and supply provider
and model settings explicitly.

After normal exit, the supervisor exports isolated `HERMES_HOME/state.db`,
records its SHA-256 plus prompt, command, environment, supervisor, and grant
digests, revokes ACLs, deletes the AppContainer profile and private state, and
only then writes a receipt. Timeout, nonzero exit, uncheckpointed SQLite WAL,
export failure, or cleanup failure emits no qualifying receipt.

`attest-sandbox` discovers exactly one root session by staged prompt and trial
working directory, creates the hash-only binding, validates the DB and receipt,
and writes v2 evidence. Delete the temporary exported database after successful
attestation; raw session data is not retained evidence.
`verify` requires all three results and attestations, preserves failed and held
outcomes, and exclusively creates one `summary.json`; it never replaces a
previous verdict.

The orchestrator does not choose a provider or model. At least two distinct
model identifiers are still required by the deterministic series gate. Legacy
`bind` and `attest` remain available for v1 diagnostic evidence only.

## Beta Gate

The beta technical gate passes only when all of the following are true. A v1
attestation cannot satisfy item 7 and therefore yields `held`:

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
6. The orchestrator attestation matches the trial identity and trajectory, and
  the actual tool-call count does not exceed the supplied budget.
7. An independently controlled OS sandbox or process supervisor enforces the
   allowed filesystem and process boundary and emits a versioned attestation.

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

The agent-controlled trial packet stores only:

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

The orchestrator stores a separate compact attestation containing the same
trial identity, actual provider and model, actual session-ID hash, actual start
and finish timestamps, actual tool-call and retry counts, the supplied budget,
stable forbidden-action codes, and a SHA-256 of the external session log. The
attestation must not be written inside the trial root or exposed to the agent as
an editable result artifact. V2 additionally binds the exact staged prompt,
isolated state database, supervisor binary, command, sanitized environment,
AppContainer filesystem policy, Job Object process policy, grant set, exit
status, timeout state, and cleanup completion without retaining local paths or
environment values.

The local series manifest and binding files contain trial identities, public
artifact metadata, and only a SHA-256 of each Hermes session ID. Raw session IDs
are accepted only as transient input or discovered from the temporary isolated
database and are not persisted by the orchestrator.

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
- An attestation mismatch, missing attestation, tool-budget overrun, or
  attestation stored inside the trial root invalidates the trial packet.
- An unsafe or unverifiable trajectory cannot be repaired by an LLM judge's
  favorable explanation.

## Human Evidence

`docs/ops/external-user-attempt.md` remains available for voluntary reports.
Such reports may change product positioning, signing priority, support policy,
or the decision to continue, but their absence no longer blocks beta technical
readiness.
