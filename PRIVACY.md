# Privacy

## Velox Tooling

The Velox CLI and native host do not send telemetry, crash reports, analytics,
or automatic update requests. The build path reads the project manifest,
static assets, and prebuilt host, then writes local build outputs.

## Local Runtime Data

The installed WebView2 Runtime can store cookies, cache, local storage,
IndexedDB, and other browser-profile data in an application-specific local user
data directory. Velox does not upload that data. Removing a portable app does
not automatically remove its WebView2 profile.

## Packaged Applications

An application packaged with Velox can implement its own network requests,
analytics, accounts, or data storage. Those flows belong to that application
and require its own privacy disclosure. They are not Velox data collection.

## Maintainer Services

GitHub Actions processes source and build evidence for repository automation.
The unsigned developer-preview workflow does not contact a signing provider.
If ADR 0011 later reactivates SignPath, it will process only the two release
executables and signing metadata after provider onboarding is approved. These
maintainer services do not receive end-user application data from the Velox
runtime.

## Clean-Room Agent Evaluation

ADR 0018 permits maintainer-orchestrated LLM agent trials for beta technical
readiness. Public evidence may retain provider and model identifiers, a SHA-256
hash of the session identifier, release and artifact hashes, redacted Windows
and WebView2 versions, command classes, counts, stable diagnostics, and relative
artifact paths.

Evaluation evidence must not retain provider credentials, raw session tokens,
full transcripts, chain of thought, local absolute paths, usernames,
environment variables, proprietary application data, screenshots containing
personal information, or raw crash dumps. Trial applications use synthetic
task data only. Passing an agent evaluation is not a human adoption claim.

The orchestrator may retain a compact attestation outside the agent-controlled
trial workspace. It contains only the hashed session identifier, redacted
timing and count metadata, stable forbidden-action codes, and a SHA-256 binding
to the external session log; the raw log is not copied into the public packet.
The Hermes adapter computes that binding from a canonical projection of the
selected session and message rows in memory. Raw prompts, reasoning, tool
arguments, tool results, the raw session ID, and the Hermes database are not
written to the attestation output.
The series orchestrator also stores only a SHA-256 session binding. The raw
Hermes session ID is transient command input and is not written to the series
manifest, generated prompt, binding, attestation, summary, or public packet.
