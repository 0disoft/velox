# Validation

- Status: M2 complete; M3 active with the Wails cold-build gate passed

## Validation Source of Truth

This document owns stable validation names for this scaffold.

## Standard Validation Names

- format
- lint
- typecheck
- test
- contract
- migration-check
- smoke
- docs
- check

## Required Final Report

Final responses must list executed validations, passed validations, skipped validations, skip reasons, and remaining risk.

## Runner Policy

Task runner files are optional. This repository still uses runner `none`.
The parent workspace command contract currently provides these bounded intents:

- `velox_format` maps to format.
- `velox_lint` maps to lint.
- `velox_test` maps to test.
- `velox_build` maps to the production Go host build.
- `velox_release_bundle` builds the Go CLI and host and assembles the unsigned,
  deterministic Windows x64 release bundle.
- `velox_consumer_build_smoke` invokes only the assembled release CLI, creates
  a dependency-free starter, diagnoses its platform, WebView2, project, and
  bundled-host compatibility, builds it twice, checks
  byte-identical archive hashes, and inspects both the portable directory and
  ZIP.
- `velox_cli_run_smoke` launches source assets through the assembled release
  CLI, requires the host to reach its ready callback, exits it, and verifies the
  temporary runtime configuration was removed.
- `velox_consumer_benchmark_smoke` runs three local samples to validate the
  benchmark harness and schema without turning unavailable process tracing into
  a false pass.
- `velox_consumer_benchmark` runs ten local clean-output samples and enforces
  build-duration, cache, intermediate-file, and compiler/package-manager
  child-process gates. It is expected to fail when Windows process-start
  tracing is unavailable.
- `velox_consumer_e2e_smoke` validates release extraction, initialization,
  build, inspection, success/failure result serialization, and the end-to-end
  JSON Schema using a local release ZIP. Its result is not hosted cold-build
  evidence. Child-process tracing may remain `unverified` locally.
- `velox_consumer_e2e_failure_smoke` injects a release-checksum mismatch and
  requires a schema-valid `release-verification` failure result.
- `velox_consumer_e2e_summary_smoke` aggregates one local raw result and
  validates the summary schema without promoting it to hosted evidence.
- `velox_consumer_e2e_summary_failure_smoke` aggregates one success and one
  injected failure, requires the summary command to fail, and verifies the
  failed sample remains in the written summary.
- `velox_consumer_e2e_hosted_gate_smoke` simulates bounded hosted metadata and
  requires unavailable process tracing to preserve a raw result while failing
  the hosted evidence gate.
- `velox_consumer_e2e_hosted_summary_gate_smoke` requires an unverified hosted
  process trace to remain counted and fail the aggregate summary gate.
- `velox_workflow_validate` parses the repository-owned GitHub Actions workflow
  with `yq` without modifying it.
- `velox_startup_smoke` maps to smoke.

The hosted `Consumer evidence` workflow additionally runs three startup
lifecycle samples for pull requests and `quick` manual dispatches. A `full`
manual dispatch, weekly schedule, or release-candidate tag runs ten lifecycle
and ten consumer samples. It validates
`velox.startup-lifecycle/v3`, derives and validates
`velox.startup-lifecycle-summary/v1` plus
`velox.startup-lifecycle-phase-summary/v1`, and uploads all results with
`always()`. The phase summary computes interval p50 and p95 values and the
dominant immediate-startup interval directly from raw v3 evidence.
Lifecycle v3 preserves the host-local startup and shutdown phase timelines for
both the first launch and the immediate same-profile relaunch.
This longer evidence path is intentionally separate from the local one-sample
`velox_startup_smoke` intent.

An explicit manual `include_profile_comparison` input runs three alternating,
serial same-profile versus fresh-profile pairs and validates
`velox.startup-profile-comparison/v1`. It is disabled for ordinary pull request,
scheduled, and release-candidate evidence.

Each weekly schedule also builds `velox.startup-history/v1` from the current
lifecycle summary and up to eleven prior successful scheduled artifacts. The
history is grouped by runner image version and WebView2 version, retained for
90 days, and remains diagnostic evidence rather than an automatic regression
gate. Manual runs can exercise the same collector with
`include_startup_history`.

The `Actions warning monitor` workflow allocates a runner after scheduled or
release-candidate consumer evidence, or for an explicit manual run ID. Pull
request and ordinary manual consumer evidence produce only a skipped monitor
job. The monitor scans the bounded workflow-log archive for the known
`actions/download-artifact` `DEP0005 Buffer()` warning. It validates and uploads
`velox.actions-warning-monitor/v1`. Presence is diagnostic rather than a failed
product check; malformed or inaccessible log evidence still fails the monitor.
The platform-independent scanner uses the pinned `ubuntu-24.04` runner.

The C++23/Pixi M0 reference intents were retired after ADR 0005 selected Go
for both production executables. Historical comparison results remain in ADR
0004 and the performance budget.

Unconfigured validation names remain skipped and must not pass with a fake
success.

## M2 Security Evidence

| Contract | Executable evidence |
| --- | --- |
| Trusted virtual origin and top-level messages | `internal/webview2` origin tests and Windows startup security fixture |
| Navigation, frame, popup, download, and permission denial | Windows startup security fixture policy audit |
| Closed method and permission table | `internal/ipc` dispatcher tests |
| Payload, nesting, request ID, duplicate, and in-flight limits | `internal/ipc` malformed and concurrency tests |
| Frozen JavaScript bridge | embedded bridge contract test and Windows startup IPC invocation |
| Production development-tool restrictions | runtime security source guard and startup production path |
| No listening socket or broad native API | production-host source guard and closed dispatcher tests |
| Missing runtime and malformed configuration | startup and runtime-configuration failure tests |
| Path, archive, staging, and release checksum controls | asset, build-plan, builder, inspector, host metadata, and release tests |

The security fixture must complete both a trusted `app.getInfo` invocation and
the five browser-policy denials before emitting `security-ok`.

## Hygiene Validation

Repository hygiene file changes must check line-ending churn, binary diff pollution,
tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

general validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Repository Shape

cli-tool validation must stay repository-shape focused and must not imply generated application source code.
