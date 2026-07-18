# Roadmap

- Status: Draft
- Owner: Project maintainer

## Roadmap Rule

Milestones are evidence gates, not dates. A later milestone does not start
until the previous milestone's exit criteria are met. Features that do not
improve or protect the headline metrics and runtime guardrails stay deferred.

## M0: Feasibility and Kill Test

Status: Complete.

### Deliver

- Minimal pure-Go Windows x64 WebView2 host.
- A bounded C++23 reference used only for the completed host-selection decision.
- One dependency-free hello fixture.
- A benchmark-only ready marker.
- Fresh-profile and warm-profile startup harness.
- A written comparison with Wails and the closest compile-free alternative.

### Exit criteria

- Both hosts render the same fixture and shut down reliably.
- Raw process-to-ready measurements are reproducible.
- The Go host meets the ADR-0001 startup and maintainability gate.
- The compile-free packaging hypothesis remains meaningfully distinct from a
  PWA and existing wrappers.

### Stop condition

Stop implementation if the product has no defensible advantage beyond omitting
features or if WebView2 lifecycle cannot be implemented safely with the
selected host approach.

## M1: Compile-Free Vertical Slice

Status: Complete.

### Deliver

- Go CLI with init, validate, doctor, run, build, inspect, and version.
- Versioned manifest and build-result schemas.
- Unchanged prebuilt generic host.
- External runtime configuration and static asset directory.
- Atomic output staging.
- Deterministic portable ZIP.
- Dependency-free hello example.

### Exit criteria

- A clean Windows x64 machine builds the example without a compiler or Node.js.
- Build works offline after acquiring one pinned Velox release bundle.
- Consumer Actions cache upload is zero bytes.
- Repeated equivalent builds produce identical unsigned ZIP digests.
- Failure preserves source and prior successful output.

## M2: Minimum Security Contract

Status: Complete.

### Deliver

- Trusted top-level origin checks.
- Remote navigation, popup, download, and browser-permission denial.
- Closed permission and native method tables.
- IPC size, nesting, and in-flight limits.
- Production development-tool restrictions.
- Path, link, reparse-point, reserved-name, and archive-entry validation.
- Threat model and adversarial tests.

### Exit criteria

- Every security rule in the product specification has an executable test.
- Malformed configuration and messages fail closed.
- No filesystem, shell, process, sidecar, or plugin capability exists.

## M3: Public Benchmark

Status: Complete.

The Wails zero-cache cold-build gate is complete. Its machine-generated
evidence and scope limits are recorded in
`docs/engineering/03-performance-budget.md`. The pairwise result did not by
itself complete M3; the deliverable audit below records the additional evidence
that closed the milestone.

### Deliverable audit

Audit baseline: `velox-bench` revision
`95ffd8b38cbadf44cd681a55efb56bba7d30649c`.

| Deliverable | State | Current evidence or gap |
| --- | --- | --- |
| Separate public benchmark repository | Complete | `0disoft/velox-bench` owns the public contracts, fixtures, workflows, and raw-evidence schemas |
| Pinned Velox, Wails, Neutralino, and Tauri adapters | Complete | `bench.lock.json` pins immutable revisions and the contract check enforces all four adapters and byte-identical hello assets |
| Hello and deterministic asset-pack fixtures | Complete | The dependency-free hello fixture remains canonical; the asset-pack manifest pins a dependency-free 1,000-file, exact-10-MiB generator contract and tree digest without committing generated payloads. [Run 29627187122](https://github.com/0disoft/velox-bench/actions/runs/29627187122) completed one non-publishable hosted sample for all four adapters; [run 29627976497](https://github.com/0disoft/velox-bench/actions/runs/29627976497) then verified Velox declared-ZIP accounting at zero surviving intermediate files and bytes |
| Zero-cache and recommended-cache suites | Complete | The hosted zero-cache suite is published. [Recommended-cache run 29631255241](https://github.com/0disoft/velox-bench/actions/runs/29631255241) completed schema-valid Velox-Wails prime and warm evidence on separate runners, recorded exact GitHub cache archive bytes, retained `comparativeClaimAllowed: false`, and left no run-owned cache entry after cleanup |
| Raw versioned JSON results and generated summary tables | Complete | The pinned pair evidence, normalized run metadata, publication contract, and README table are committed; contract checks regenerate the publication and table in memory to reject hand-edited values |
| CI resource usage disclosure | Complete | The publication reports workflow wall time, aggregate observed job runtime, job outcomes, artifact count and bytes, and cache upload while explicitly separating those observations from billed Actions minutes |

All named M3 deliverables now have hosted evidence. This does not turn a
one-sample recommended-cache diagnostic into a comparative product claim; the
published zero-cache pair result remains the performance source of truth.

### Deliver

- Separate public benchmark repository.
- Pinned Velox, Wails, Neutralino, and Tauri adapters.
- Hello and deterministic asset-pack fixtures.
- Zero-cache and recommended-cache suites.
- Raw versioned JSON results and generated summary tables.
- CI resource usage disclosure.

### Exit criteria

- End-to-end setup and build phases are visible.
- p50, p95, failures, and environment metadata are published.
- Results can be reproduced from a clean repository checkout.
- Marketing language does not exceed measured evidence.

### Go-or-kill gate

- [x] Consumer cache upload is exactly zero bytes for the publishable Wails
  pair.
- [x] Velox end-to-end cold build is at least 3x faster than the pinned Wails
  fixture.
- [x] Velox remains structurally simpler than the closest compile-free
  comparison within the portable static-app boundary defined by ADR 0008.
- [x] Startup is removed as a headline advantage by ADR 0009 and retained as a
  release guardrail.

## M4: Alpha Distribution

Status: Active.

The repository now owns an unsigned evidence pipeline. It produces two
independent release builds, checks byte identity, emits checksums, a file-level
SPDX SBOM, and an unsigned in-toto/SLSA provenance statement, then runs a
checkout-free consumer build. This is release evidence, not a published alpha.
[Hosted run 29631165931](https://github.com/0disoft/velox/actions/runs/29631165931)
completed both jobs, and the downloaded artifacts passed their checksum and
bundled consumer-result schema checks.

ADR 0010 now fixes the M4 trust design: GitHub artifact attestations authenticate
the final ZIP and SBOM, while SignPath Foundation is the conditional
Authenticode provider for the CLI and unchanged host. This is a selected design,
not provider onboarding or signed-release evidence.

The repository-owned `velox.signing-record/v1` contract, dry-run generator, and
lineage verifier now bind every unsigned, provider-output, final bundle,
manifest, checksum, and SBOM digest. Dry-run evidence is mechanically
non-publishable and does not claim signature or attestation verification.
The maintainer `prepare` command also produces the provider input from exactly
the two unsigned executables with deterministic ZIP metadata, no overwrite,
and an immediate source-digest verification pass.
The lineage verifier rejects provider output unless one directory contains only
the two expected signed executable names.
Doctor now gates the documented Windows and Evergreen WebView2 compatibility
floor instead of treating any installed runtime as supported.

### Deliver

- Checksums and software bill of materials.
- Signed Velox CLI and unchanged generic host.
- Immutable release manifest.
- Installation, compatibility, security, and limitation documentation.
- Clean-runner consumer workflow example.

### Exit criteria

- A new user can reproduce the example from published documentation.
- Release artifacts and embedded contracts agree.
- Missing WebView2 and unsupported Windows environments produce actionable
  diagnostics.
- Directory asset tampering and branding limitations are prominent.

### Remaining gates

- Obtain SignPath Foundation project acceptance and approve the exact artifact
  configuration, signing policy, publisher/profile identity, and protected
  environment.
- Submit the repository-generated signing input through the accepted provider
  configuration and preserve the returned request identity.
- Implement real Authenticode verification and release-mode signing-record
  creation using provider output.
- Implement final artifact attestations and the immutable publication gate.
- Publish immutable compatibility and limitation notes beside a real alpha.
- Run the documented path from a repository and account not controlled by the
  implementation workflow.
- Verify the downloaded public artifact rather than a same-run workflow
  artifact.

## M5: Product Decision

Status: Not started. M3 is complete; M4 remains open.

Choose one:

- Continue toward a stable narrow packager.
- Reposition around CI packaging and reproducibility.
- Merge the useful work into an existing ecosystem.
- Stop the project.

The decision uses benchmark evidence, external user attempts, maintenance cost,
security review findings, and the strength of the PWA and Neutralino
counterarguments.

The Wails cold-build result and the two accepted M3 product decisions supply M5
inputs, not the product decision. Before M5 can start, the repository still
needs M4 distribution evidence, external user attempts, a bounded
maintenance-cost record, and a security review. ADR 0008 records the explicit
PWA and Neutralino counterarguments; user attempts must now test whether its
narrow portable-artifact boundary has real value.

## Deferred Until After M5

- Installers and automatic updates.
- Per-application executable resource patching.
- Application code signing automation.
- Sealed or embedded assets.
- Native filesystem, shell, process, plugin, or sidecar APIs.
- Frontend bundling, hot reload, and development server.
- macOS, Linux, ARM64, multi-window, tray, menu, and global shortcuts.

Each deferred item requires a new ADR, measured impact on the headline metrics
and guardrails, and a clear reason it belongs in core rather than an external
tool.
