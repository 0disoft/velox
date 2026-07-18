# Operability and Failure Standard

- Status: Draft
- Owner: Project maintainer

## Scope

Actutum is a local CLI and desktop host, not a hosted service. Its first
operability surface is deterministic diagnostics, artifact inspection, and
clean failure recovery.

There is no production telemetry, tracing backend, dashboard, pager, or remote
control plane.

## Failure Principles

- Fail before mutation when validation can detect the problem.
- Fail closed at trust and compatibility boundaries.
- Preserve source and previous successful output.
- Remove only temporary paths owned by the current operation.
- Return one stable process exit category and a precise diagnostic code.
- Never download a fix or dependency implicitly.
- Never hide a failed first attempt behind an automatic retry.

## Failure Categories

- CLI usage and configuration.
- Manifest syntax, schema, and semantics.
- Asset and path safety.
- Host and contract compatibility.
- WebView2 prerequisite and startup.
- Packaging, staging, archive, and promotion.
- Runtime IPC and permission denial.
- Shutdown and resource release.
- Internal invariant failure.

## Local Diagnostics

Diagnostics include:

- Stable code and category.
- Short actionable message.
- Project-relative location when available.
- Structured safe facts.
- Compatibility and artifact versions when relevant.

Diagnostics exclude secrets, source contents, ambient environment dumps,
native stack traces in normal mode, and unstable progress prose.

## Logs

Normal operation does not require a persistent log file.

Explicit verbose or diagnostic modes may emit bounded local logs. The exact
file location, retention, and redaction implementation remain UNDECIDED until
the host exists.

Logs are not uploaded automatically.

## Recovery

### Build

Delete the current staging directory, preserve prior output, and rerun only
after the diagnostic is resolved.

### Runtime

Close the failed process and correct the local prerequisite, configuration, or
asset problem. The host does not enter a degraded mode with weakened security.

### Release

Use a previously published immutable artifact when a new release is defective.
There is no database or remote state rollback.

## Health and Inspection

- doctor reports local prerequisites and compatibility without writes.
- validate reports project problems without creating output.
- inspect reports artifact versions, permissions, counts, and digests without
  execution.
- version reports supported contract ranges.

These commands are planned contracts until implementation exists.

## Release Blockers

- A common failure has no stable diagnostic.
- Build failure can damage source or prior output.
- Startup failure leaves a background process or locked resource.
- A compatibility failure is treated as a warning and execution continues.
- Required troubleshooting depends on remote telemetry.
- Failure output exposes sensitive or machine-specific data unnecessarily.
