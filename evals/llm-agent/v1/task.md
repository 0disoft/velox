# Velox Clean-Room Agent Task v1

## Evaluator Role

You are a coding agent evaluating a Windows desktop packaging tool you have not
used before. Start in a fresh Windows workspace with no prior conversation,
memory, repository checkout, or unpublished project context.

Do not expose hidden reasoning. Report observable actions, artifacts, checks,
failures, and remaining uncertainty only.

## Supplied Inputs

The orchestrator supplies exactly these values:

- `RELEASE_TAG`
- `RELEASE_URL`
- `RELEASE_SHA256`
- `RESULT_DIRECTORY`
- `SERIES_ID`
- `SEQUENCE`

You may read the public release page and public documentation reachable from
the `0disoft/velox` repository. You may not clone or download the Velox source
tree, read a maintainer workspace, use this repository's local build output, or
ask the maintainer for interactive hints after the trial starts.

Treat the public `docs/QUICKSTART.md` as the discovery entrypoint. Do not use a
local copy supplied from a maintainer workspace.

## Goal

Using only the published Velox release, create and package a local-first desktop
application named **Focus Ledger**.

Use application ID `dev.velox.agent.focusledger` and version `0.1.0` so the
packaged build result can be checked without interpreting prose.

The application must let a user:

- add a task with non-empty text;
- mark a task complete or incomplete;
- delete a task;
- see active, completed, and total counts;
- retain tasks after the application is closed and reopened;
- use the application without an external network request.

Use static HTML, CSS, and JavaScript. Do not install or invoke Go, Rust, C,
C++, Zig, Node.js, Bun, npm, pnpm, Yarn, another package manager, a frontend
bundler, or an application-specific native backend.

## Required Evaluation Path

1. Download the exact release from `RELEASE_URL` and verify its ZIP SHA-256
   equals `RELEASE_SHA256` before executing it.
2. Discover the public usage path from the release page and public repository
   documentation. Do not use unpublished instructions.
3. Initialize a project, inspect the generated contract, and author Focus
   Ledger only inside the fresh trial workspace.
4. Run the product's validation and environment diagnosis paths.
5. Build into two distinct clean output directories and verify the two portable
   ZIP files are byte-identical.
6. Inspect the packaged result through the released CLI.
7. Start the packaged application and verify the required behavior and
   persistence through observable UI state.
8. Write one JSON result conforming to
   `schema/llm-agent-evaluation-v1.schema.json` and one concise Markdown report
   under `RESULT_DIRECTORY`.

## Stop Conditions

Stop and emit `failed` or `held` evidence instead of improvising when:

- the release checksum differs;
- the public documentation does not expose a usable next step;
- the supported WebView2 runtime is unavailable;
- the workflow appears to require a forbidden compiler, runtime, package
  manager, source checkout, hidden native API, or maintainer hint;
- deterministic output, inspection, startup, or required application behavior
  cannot be verified;
- a security or privacy boundary is unclear.

Do not weaken the application requirements, silently add a toolchain, or report
success from a plausible-looking final answer.

## Evidence Rules

- Final environment state and artifact hashes outrank the agent's narrative.
- Record command classes, counts, stable diagnostic codes, relative artifact
  paths, and SHA-256 values. Do not store a full transcript or chain of thought.
- Copy both build archives, the inspected `build-result.json`, and the concise
  report into `RESULT_DIRECTORY`; record each relative path and SHA-256.
- Preserve the supplied `SERIES_ID` and `SEQUENCE`. Never replace a failed or
  held sequence with a later successful trial.
- Do not include local absolute paths, usernames, tokens, environment variables,
  proprietary data, screenshots containing personal information, or raw crash
  dumps.
- `maintainerOrchestrated` must remain `true`, `externalHuman` must remain
  `false`, and `humanAdoptionClaim` must remain `false`.
- A canceled or unverifiable UI step is not a pass.

## Completion Contract

The trial passes only when every hard gate in the result schema is true, the
first and second build hashes match, the packaged application reaches usable
content, the Focus Ledger behavior is observed, no forbidden action occurred,
and `failure` is `null`.

The evaluator does not decide whether Velox enters beta. The maintainer applies
the repository's multi-trial gate after validating all trial records.
