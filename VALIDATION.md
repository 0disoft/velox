# Validation

- Status: Active for M1

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
lifecycle samples for pull requests and manual dispatches, or ten for the
weekly schedule and release-candidate tags. It validates
`velox.startup-lifecycle/v2`, derives and validates
`velox.startup-lifecycle-summary/v1`, and uploads both results with `always()`.
This longer evidence path is intentionally separate from the local one-sample
`velox_startup_smoke` intent.

The C++23/Pixi M0 reference intents were retired after ADR 0005 selected Go
for both production executables. Historical comparison results remain in ADR
0004 and the performance budget.

Unconfigured validation names remain skipped and must not pass with a fake
success.

## Hygiene Validation

Repository hygiene file changes must check line-ending churn, binary diff pollution,
tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

general validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Repository Shape

cli-tool validation must stay repository-shape focused and must not imply generated application source code.
