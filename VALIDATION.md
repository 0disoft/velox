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
- `velox_startup_smoke` maps to smoke.
- `velox_cpp_build` maps to the C++23 reference build.
- `velox_cpp_startup_smoke` maps to the C++23 startup smoke.
- `velox_startup_benchmark` maps to the repeated Go/C++23 comparison.

The C++ and Pixi intents validate reference evidence only. They are not
required after CLI packaging changes that do not touch the reference host,
benchmark harness, or reference toolchain.

Unconfigured validation names remain skipped and must not pass with a fake
success.

## Hygiene Validation

Repository hygiene file changes must check line-ending churn, binary diff pollution,
tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

general validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Repository Shape

cli-tool validation must stay repository-shape focused and must not imply generated application source code.
