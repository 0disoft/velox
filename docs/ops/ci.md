# CI

- Status: Pre-implementation
- Owner: Project maintainer

## Current State

No CI workflow or executable project runner exists. The following stages are
the required design for M0 and later, not current behavior.

## Planned Pull-Request CI

- Documentation and contract consistency.
- Go formatting, static analysis, unit tests, and contract tests after source
  exists.
- Windows x64 host build.
- Dependency-free hello build and startup smoke.
- A bounded Velox-only performance smoke sample.
- Artifact and generated-output drift checks.

The full cross-framework benchmark matrix does not run on every pull request.

## Planned Scheduled and Release CI

- Reproducibility across clean workspaces.
- Fresh and warm startup measurements.
- Zero-cache and recommended-cache benchmark suites.
- Wails, Neutralino, and Tauri comparison adapters.
- Software bill of materials and release checksum checks.
- Release artifact smoke and inspect.

## Cache Policy

### Consumer example

The documented consumer workflow uses no GitHub Actions cache.

### Maintainer CI

Maintainer toolchain caches may be considered only when their byte size,
restore time, invalidation, and cost are measured. They must never be presented
as consumer cache requirements.

### Benchmark

Zero-cache and recommended-cache results remain separate. Benchmark caches use
bounded unique keys and cleanup so the benchmark repository does not consume
unbounded storage.

## Job Isolation

- Framework comparisons run in separate jobs.
- Runner image, framework version, action revision, fixture digest, and
  WebView2 version are recorded.
- Mutable latest labels and versions are not the only record of an official
  result.
- Failed and timed-out jobs remain part of benchmark evidence.

## Artifacts

Pull requests retain only bounded diagnostic and smoke artifacts. Raw benchmark
retention and release artifact retention remain UNDECIDED until workflows are
implemented.

Compiler caches, package-manager caches, and workspaces are not uploaded as
ordinary artifacts.

## Failure Policy

- Do not retry a failure and report only the green attempt.
- Preserve the first failure classification and relevant bounded artifact.
- Performance regressions follow the noise and threshold rules in the
  performance budget.
- A missing configured check is a release blocker, not an implicit pass.

## Branch Protection

Required check names and branch rules remain UNDECIDED until actual workflow
jobs exist. Documentation must not claim branch protection before repository
settings are verified.
