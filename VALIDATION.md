# Validation

- Status: Active for M0

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
- `velox_build` maps to the M0 host build.
- `velox_startup_smoke` maps to smoke.
- `velox_cpp_build` maps to the C++23 reference build.
- `velox_cpp_startup_smoke` maps to the C++23 startup smoke.
- `velox_startup_benchmark` maps to the repeated Go/C++23 comparison.

Unconfigured validation names remain skipped and must not pass with a fake
success.

## Hygiene Validation

Repository hygiene file changes must check line-ending churn, binary diff pollution,
tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

general validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Repository Shape

cli-tool validation must stay repository-shape focused and must not imply generated application source code.
