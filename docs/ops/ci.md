# CI

- Status: Consumer evidence workflow active
- Owner: Project maintainer

## Current State

`.github/workflows/consumer-evidence.yml` builds one unsigned Windows x64
release artifact, then passes that exact ZIP to isolated consumer jobs. Pull
requests run one consumer contract sample. Manual dispatches expose a bounded
`quick` or `full` evidence tier: quick runs one consumer sample and three
lifecycle samples, while full runs ten of each. The weekly schedule and
release-candidate tags use the full tier.

Manual dispatch also exposes a disabled-by-default profile comparison. When
selected, the producer runs three paired same-UDF and fresh-UDF relaunch trials,
validates `velox.startup-profile-comparison/v1`, and retains the raw comparison
for 30 days. It is diagnostic evidence and does not expand the normal CI path.

After consumer jobs finish, an always-run summary job downloads every available
raw result, rejects duplicate sample IDs, preserves failures and missing sample
counts, and calculates minimum, p50, p95, and maximum over successful samples.
The summary job fails when a sample is missing, failed, or points at a different
release digest.

Each consumer build traces child-process starts. Hosted samples are rejected
unless the measured CLI process is identified and no compiler or package
manager descendant appears. The tracer prefers WMI or CIM process-start events
and falls back to a non-administrator Win32 Toolhelp snapshot poller when event
subscriptions are unavailable. Failure of every backend is `unverified`, not a
pass, and makes hosted evidence fail. The trace closes immediately after the
single measured `velox build` process; later inspection commands are excluded.

The consumer clock starts after checkout and before artifact download. It ends
after release extraction, dependency-free project initialization, build, and
portable ZIP inspection. Maintainer compilation happens in a different job and
is not included in the consumer path.

## Pull-Request CI

- Documentation and contract consistency.
- Go formatting, static analysis, unit tests, and contract tests after source
  exists.
- Windows x64 host build.
- Dependency-free hello build and startup smoke.
- One bounded Velox-only end-to-end contract sample.
- Three serial fresh/immediate same-profile startup lifecycle samples.
- Artifact and generated-output drift checks.

The full cross-framework benchmark matrix does not run on every pull request.

## Scheduled and Release CI

- Reproducibility across clean workspaces.
- Ten isolated consumer end-to-end samples.
- The Windows producer job records ten fresh/immediate same-profile startup
  lifecycle samples, validates `velox.startup-lifecycle/v2`, derives a
  `velox.startup-lifecycle-summary/v1` correlation and ordering summary, and
  preserves both as a 30-day artifact even when a sample fails. Release
  candidate tags use the same ten-sample lifecycle path. Cross-framework
  startup comparison remains pending.
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

The intermediate unsigned release artifact is retained for one day. Raw
consumer result JSON is retained for seven days. Failed measurement jobs upload
their structured failure result when the script reached result serialization.
The generated consumer and startup lifecycle summaries are retained for 30
days.

The workflow pins checkout and artifact actions to immutable commit SHAs. It
also pins `setup-go`, reads the Go version from `go.mod`, and disables its
built-in cache. It does not use `actions/cache`. The release ZIP is uploaded
without recompression because it is already compressed.

Dependabot checks the `github-actions` ecosystem weekly and opens reviewable
pull requests without auto-merge. The workflow also runs
`cmd/velox-action-pins`, which rejects mutable `actions/*` references, stale
stable-release comments, and SHAs that do not match the official release tag.
Independent action repositories are checked concurrently while output and
failure ordering remain deterministic.

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
