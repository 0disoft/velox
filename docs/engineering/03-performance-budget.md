# Performance Budget

- Status: Draft
- Owner: Project maintainer

## Priority

Velox optimizes, in order:

1. End-to-end cold build time.
2. Consumer GitHub Actions cache upload.
3. Process-to-ready application startup.

Artifact size and memory use are collected as guardrails but are not headline
metrics unless product scope changes.

## Measurement Profiles

### End-to-end cold build

Elapsed time from a completed source checkout through acquiring the pinned
framework release and producing the final portable ZIP. Toolchain and package
installation are included.

### Local clean-output build command

Elapsed time from starting the already-installed build command through final
ZIP completion. The local harness reuses one initialized project and gives each
sample a new output root. It does not reset the OS file cache, so this profile
must not be called a cold build.

### Fresh-profile startup

Elapsed time from process creation until the application emits the ready marker
after DOMContentLoaded and two animation frames, using a new WebView2 user-data
directory.

### Warm-profile startup

The same ready measurement after five unrecorded warmups while reusing the
WebView2 user-data directory. A settled warm run and an immediate relaunch are
different lifecycle conditions and must be reported separately after M0. A
delay caused by the previous browser process releasing the profile is not
discarded as an outlier.

### Cache footprint

Two values are reported separately:

- Bytes explicitly uploaded to GitHub Actions cache.
- Increase in framework-specific tool and build cache directories during the
  job.

## Provisional MVP Gates

These are go-or-kill targets, not published performance claims.

| Metric | Gate |
| --- | --- |
| Consumer Actions cache upload | Exactly 0 bytes |
| Consumer native compiler execution | None |
| Consumer Node.js execution | None |
| Surviving intermediate files | 0 files outside declared output |
| Hello local clean-output build command | p95 at or below 2 seconds on the pinned runner |
| End-to-end cold build | At least 3x faster than the pinned Wails fixture |
| Go host startup | Record regressions against its pinned Go baseline; C++23 remains diagnostic only |
| Startup claim | Publish only when the advantage exceeds noise and 10% |

If the Go host exceeds its startup allowance, investigate the repository-owned
adapter and WebView2 lifecycle before changing languages. ADR 0005 permits
reopening the language decision only if a bounded pure-Go adapter cannot safely
represent the required COM lifecycle or WebView2 security controls.

## Benchmark Rules

- Pin the runner image, framework versions, action revisions, fixture digest,
  and Velox artifact checksum.
- Run each framework in an isolated job.
- Use identical application assets except for the smallest required ready
  adapter.
- Report at least p50, p95, minimum, maximum, failures, and timeouts.
- Preserve raw machine-readable results.
- Separate zero-cache and framework-recommended-cache suites.
- Include release download and setup in the end-to-end headline.
- Record CPU, memory, runner image, Windows version, WebView2 version, and
  artifact digest.
- Never delete a failed run from the published sample.
- Do not call a result faster when the difference is below 10% or compatible
  with observed noise.

## Fixtures

### Hello

A dependency-free HTML, CSS, and JavaScript application with no external
network, font, or framework dependency.

### Asset pack

A deterministic fixture containing many small files and approximately 10 MiB
of static assets. The exact file count, seed, and digest remain UNDECIDED until
the benchmark repository is created.

## CI Frequency

- Pull requests run correctness checks and a small Velox-only smoke sample.
- Scheduled and release-candidate workflows run the full cross-framework
  matrix.
- Full benchmark repetitions do not run on every commit because the benchmark
  itself must not consume disproportionate CI resources.

## Consumer Build Harness

`scripts/measure-consumer-build.ps1` owns the local Windows measurement path.
It initializes one dependency-free project and gives each repetition a clean
output root. Build-command duration excludes initialization, inspection, schema
validation, and process-trace draining.

The harness records controlled local observations under
`velox.consumer-benchmark/v1`, validates them against
`schema/consumer-benchmark-v1.schema.json`, and reports:

- minimum, p50, p95, and maximum build-command duration;
- portable output and archive bytes and digests;
- Velox-owned cache-directory growth;
- source-tree changes and surviving intermediate files;
- descendants of the measured CLI process matching compiler or package-manager
  executable names.

Process tracing prefers Windows WMI or CIM process-start events. If the runner
denies those subscriptions, a non-administrator Win32 Toolhelp snapshot poller
samples process identity and parent relationships every two milliseconds. If
all trace backends are unavailable, that gate is `unverified`, never `pass`.
The trace covers exactly one measured `velox build` process and closes before
artifact inspection or version queries. The three-sample smoke preserves any
diagnostic result, while the ten-sample gate requires every gate to pass.

`workflowDeclaredActionsCacheUploadBytes` is a workflow-contract field, not a
local measurement. A future hosted workflow must contain no cache action and
must preserve the workflow source with the raw result before this field
supports a public cache claim.

The result contract records the monotonic clock, exact timing boundaries,
excluded work, zero warmups, serial concurrency, fixture digest, output state,
and uncontrolled OS file-cache caveat. Local samples are directional evidence;
only isolated hosted jobs may support an end-to-end cold-build claim.

## Consumer End-to-End Harness

`scripts/measure-consumer-e2e.ps1` consumes an already-built release ZIP. Its
hosted timing boundary starts after checkout and immediately before GitHub
artifact acquisition. It includes artifact transfer, release verification and
extraction, project initialization and validation, one build, and final ZIP
inspection. Maintainer compilation runs in a separate producer job.

Successful and failed observations use `velox.consumer-e2e/v1`. A failure
records its phase and returns non-zero without copying raw exception text or
local paths into the result. Local invocations are labeled
`local-contract-smoke`; only the isolated `windows-2025` consumer jobs are
`hosted-runner-evidence`.

The end-to-end clock uses UTC wall time because the boundary crosses GitHub
Actions steps and processes. The nested build-command duration still uses a
monotonic `Stopwatch`. Results must therefore retain both clocks and must not
compare local smoke duration with hosted evidence.

`scripts/summarize-consumer-e2e.ps1` validates every raw result before
aggregation. It reports expected, observed, successful, failed, and missing
sample counts; rejects duplicate sample IDs; requires one release archive
digest; and calculates nearest-rank p50 and p95 only from successful samples.
The summary still records failed samples and returns non-zero when evidence is
incomplete.

The end-to-end harness subscribes to Windows process-start events around the
consumer build. It records process names only, never arguments or environment
values. Hosted evidence fails when tracing is unavailable, when the measured
CLI root process cannot be identified exactly once, or when a compiler or
package-manager descendant is observed. Local permission failures remain
`unverified` and cannot support a public compiler-free claim.

## Regression Policy

Before a baseline exists, changes must report that performance is unmeasured.
After a baseline exists:

- A 15% or greater p50 cold-build regression blocks release.
- A 10% or greater p95 startup regression requires investigation.
- Any non-zero consumer Actions cache upload blocks release.
- Any new compiler or package-manager requirement blocks release.

Small hosted-runner variations do not block a pull request without repeated
evidence.

## Ready Marker

The common application marker must:

1. Wait for DOMContentLoaded.
2. Wait for two requestAnimationFrame callbacks.
3. Notify a benchmark-only native channel.

Window creation alone is never the ready event.

## Current Evidence

The latest local comparison used ten fresh runs and ten immediate same-profile
runs after five warmups per host. It measured process creation through the
shared fixture's DOMContentLoaded-plus-two-animation-frame marker after the Go
security adapter was enabled.

| Host | Distributed native files | Fresh p50 | Fresh p95 | Immediate warm p50 | Immediate warm p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Go runtime | 3,126,784 bytes | 1,048.50 ms | 1,209.65 ms | 7,076.14 ms | 7,410.56 ms |
| C++23 reference | 175,968 bytes | 944.50 ms | 1,040.39 ms | 7,004.62 ms | 7,116.60 ms |

The C++23 size includes the 11,776-byte executable and 164,192-byte
`WebView2Loader.dll`. The Go executable embeds its loader. These are local,
directional results, not a release baseline: runner and WebView2 metadata are
not yet captured and run order is fixed. Both implementations now show the
same approximately seven-second immediate-relaunch delay, so the delay belongs
to WebView2 browser-process teardown rather than the host language. Fresh Go
and C++ results differ by about 104 ms at p50; this is not a sufficient basis
for a startup marketing claim.

The Go lifecycle smoke now measures a first launch, an immediate second launch
using the same profile, ready-to-host-exit time, and final profile release. The
repeated run after explicit controller and COM cleanup showed an immediate
relaunch near 7.1 seconds. WebView2's pinned SDK contract states that `Close` is
synchronous while user-data-folder release completes only after the shared
browser process exits. This is a recorded lifecycle regression, not a startup
outlier. The provisional smoke ceiling is 10 seconds until the architecture
either avoids browser-process teardown on ordinary restart or accepts the delay
as a platform limitation. Startup remains a guardrail metric, not a product
advantage, until cross-framework evidence exceeds the documented noise gate.
