# Diagnostic Agent Session Evidence

## Scope

`velox.agent-session-evidence/v1` is a runner-neutral, diagnostic-only ledger.
The schema is in `schema/agent-session-evidence-v1.schema.json`; structural and
cross-record checks are implemented in `scripts/agent-session-evidence.ts`.
This is not an evaluator, log extractor, or a new attestation admission path.
Existing v1/v2 evaluation and ADR 0018 qualification requirements are unchanged.

## Counting Unit

- Count each issued leaf tool request once, including succeeded, failed,
  cancelled, denied, and pending requests.
- Do not count orchestration containers in addition to their children. Two
  child requests count twice, even when dispatched in one batch.
- Consolidate started, updated, completed, and replay events with the same
  invocation ID before emitting this ledger. Duplicate canonical IDs are rejected.
- A new tool retry gets a new ID and counts again. Its retry target must precede
  it, have failed, been cancelled, or been denied, and match its tool and kind.
  Input may change. Provider HTTP retries are separate from tool retries.
- One shell tool request remains one call even if it launches several processes.
  Forbidden-action auditing is separate; a tool budget is not a process budget.
- File-change events count only when they represent a distinct issued tool
  request, not a secondary observation of an already counted command.

The existing tool-call budget is 70. Over-budget ledgers remain inspectable.
The 500-record schema limit bounds diagnostic input; it is not an admission budget.

## Completeness and Provenance

Use canonical millisecond UTC timestamps and SHA-256 digests of source bytes,
session ID, prompt, invocation ID, and tool input. Keep raw prompts, commands,
outputs, session identifiers, paths, credentials, and logs outside this record.
Tool and source identifiers must be stable public names, not encoded payloads.
Digests do not guarantee anonymization of guessable inputs.

Set `observationComplete` to false for truncated logs or uncertain nested-call
coverage. It is a producer assertion, not independent attestation. Pending calls
prevent a complete count. A completed session does not prove task success.
The validator checks timestamp ordering, duplicate IDs, and retry relationships;
JSON Schema alone does not check these cross-record constraints.

Every summary retains `admission: diagnostic-only` and
`betaTechnicalGate: false`, including complete, under-budget sessions.
No route identity, fresh-session claim, clean-room isolation, enforced sandbox,
or beta qualification can be inferred from this ledger.

## Verification

The configured `velox_neutral_session_test` intent runs synthetic ledger tests
and existing evaluation regression tests offline. No live log reading, model
execution, Hermes process, route adapter, or sandbox integration is included.
