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

### Build-command cold

Elapsed time from starting the already-installed build command through final
ZIP completion on a clean project workspace.

### Fresh-profile startup

Elapsed time from process creation until the application emits the ready marker
after DOMContentLoaded and two animation frames, using a new WebView2 user-data
directory.

### Warm-profile startup

The same ready measurement after five unrecorded warmups while reusing the
WebView2 user-data directory.

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
| Hello build-command cold | p95 at or below 2 seconds on the pinned runner |
| End-to-end cold build | At least 3x faster than the pinned Wails fixture |
| Go host versus C++23 reference startup | No more than 10 ms or 10% slower at p95 |
| Startup claim | Publish only when the advantage exceeds noise and 10% |

If the Go host exceeds its startup allowance or cannot safely maintain WebView2
COM lifetimes, the fallback is a C++23 host while retaining the Go CLI and
compile-free consumer build.

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

No Velox implementation or baseline exists yet. Every numeric threshold in this
document is provisional until M0 records reproducible raw data.
