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

The weekly schedule builds `velox.startup-history/v1` from its current
lifecycle summary plus up to eleven prior successful scheduled artifacts. A
manual `include_startup_history` dispatch can exercise the same path. Points
are grouped by exact runner image version and WebView2 version so environment
changes are visible instead of being mistaken for product regressions. Missing
or expired historical artifacts remain explicit collection issues. The history
does not make an automatic pass/fail regression decision.

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
  lifecycle samples, validates `velox.startup-lifecycle/v3`, derives a
  `velox.startup-lifecycle-summary/v1` correlation and ordering summary plus a
  `velox.startup-lifecycle-phase-summary/v1` interval and dominant-phase
  summary, and
  preserves the lifecycle evidence for 90 days even when a sample fails. The
  weekly schedule also aggregates at most twelve history points. Release
  candidate tags use the same ten-sample lifecycle path without history
  aggregation. Cross-framework immediate-relaunch cause classification lives in
  the separate `velox-bench` repository. It is not a product startup ranking:
  its adapters share a bounded ready marker but not equivalent internal phase
  instrumentation, and public-alpha availability does not change that evidence
  boundary.
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
The generated consumer summary is retained for 30 days. Startup lifecycle
evidence and its optional history are retained for 90 days so twelve weekly
points remain collectible despite normal scheduling jitter.

The workflow pins checkout and artifact actions to immutable commit SHAs. It
also pins `setup-go`, reads the Go version from `go.mod`, and disables its
built-in cache. It does not use `actions/cache`. The release ZIP is uploaded
without recompression because it is already compressed.

Clarissimi is the single deliberate moving third-party Action. The contributor
recognition workflow follows the maintainer-promoted
`0disoft/clarissimi@v0` channel so fixes arrive without a repository edit. Its
pre-merge decision job is read-only and advisory by default. Merged pull
requests create only a review draft, and promotion creates a second pull
request after maintainer approval. The proposal jobs persist checkout
credentials only for their scoped branch push and do not commit directly to
`main`.

The workflow intentionally has no provider token. Its deterministic initial
draft is only an inbox scaffold: a maintainer or delegated coding agent must
replace or correct the assessment, change its approval status, and merge that
draft before dispatching `promote-draft` with the exact checked-in draft path.
After the advisory flow has been exercised, repository variable
`CLARISSIMI_GATE_MODE=required` can make the existing decision job fail closed
without renaming the check.

Dependabot checks the `github-actions` ecosystem weekly and opens reviewable
pull requests without auto-merge. The workflow also runs
`cmd/velox-action-pins`, which rejects mutable `actions/*` references, stale
stable-release comments, and SHAs that do not match the official release tag.
Independent action repositories are checked concurrently while output and
failure ordering remain deterministic.

Compiler caches, package-manager caches, and workspaces are not uploaded as
ordinary artifacts.

## Upstream Action Warning Monitor

`Actions warning monitor` allocates its `ubuntu-24.04` runner only after a
scheduled or release-candidate `Consumer evidence` run, or for an explicit
completed run ID. Pull-request and ordinary manual evidence events create a
skipped job without runner allocation. It reads the selected run's log archive
with `actions: read`, scans only for the known `actions/download-artifact`
`DEP0005 Buffer()` signature, validates
`velox.actions-warning-monitor/v1`, and retains the report for 30 days. The
workflow never logs its token and does not use `download-artifact` to inspect
itself.

`present` is a diagnostic state, not a product failure. This keeps the latest
stable immutable action pin while making the upstream runtime warning visible.
An inaccessible or malformed log archive fails the monitor because that means
there is no evidence. When the upstream action stops emitting the signature,
the report changes to `absent` without a repository change.

## Failure Policy

- Do not retry a failure and report only the green attempt.
- Preserve the first failure classification and relevant bounded artifact.
- Performance regressions follow the noise and threshold rules in the
  performance budget.
- A missing configured check is a release blocker, not an implicit pass.

## Branch Policy

The Alpha repository uses maintainer-owned direct pushes to `main`. It does not
declare a pull-request-only workflow, required status checks, or branch
protection as a product contract. Existing workflow jobs provide validation and
evidence; their presence alone does not make them merge gates.

On 2026-07-20, the GitHub repository settings showed zero classic branch
protection rules and zero rulesets. This is a point-in-time settings
observation, not an automated drift check.

Before beta or external contributors receive write access, the maintainer must
choose required checks and branch rules and then verify the corresponding
repository settings. Documentation must not claim branch protection before
that verification exists.
