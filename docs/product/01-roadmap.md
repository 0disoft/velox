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

Status: Complete.

The repository owns an unsigned evidence pipeline. It produces two
independent release builds, checks byte identity, emits checksums, a file-level
SPDX SBOM, and an unsigned in-toto/SLSA provenance statement, then runs a
checkout-free consumer build. Tag evidence run `29714104653` and publication run
`29714173324` produced the public unsigned `v0.5.10-alpha.1` prerelease from
commit `9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Public verifier run
`29715002921` independently supplied the release digest, downloaded the public
assets without checkout, and passed build, inspection, and startup-ready gates.

The separate public `0disoft/velox-consumer-smoke` repository then consumed the
pinned public release from a hosted clean runner without Velox source checkout,
a consumer compiler, Node.js, package-manager commands, or Actions cache.
[Run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250)
passed every sidecar, CLI, deterministic-build, inspection, and startup check.
Its evidence is deliberately fixed to `maintainerControlled: true` and
`externalUserAttempt: false`.

ADR 0011 now fixes the first distribution order: publish an explicitly unsigned
developer preview, collect external acquisition evidence, and treat code signing
as a later adoption-triggered trust improvement. ADR 0010 remains the design for
a future signed channel, not an M4 prerequisite.

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
The maintainer Authenticode verifier now rejects unexpected provider files,
non-valid signatures, publisher-subject drift, non-SHA-256 signatures, missing
timestamp identities, and different signer certificates. Its successful path
still needs accepted provider output and the approved publisher subject.

### Deliver

- Checksums and software bill of materials.
- Explicitly unsigned Velox CLI and unchanged generic host in a developer-preview prerelease.
- Immutable release manifest.
- Installation, compatibility, security, and limitation documentation.
- Clean-runner consumer workflow example.

### Exit criteria

- A new user can reproduce the example from published documentation.
- Release artifacts and embedded contracts agree.
- Missing WebView2 and unsupported Windows environments produce actionable
  diagnostics.
- Directory asset tampering and branding limitations are prominent.

### Completion evidence

The immutable tag, manual publication, public warnings, and no-checkout public
verification are complete for `v0.5.10-alpha.1`. ADR 0016 replaces the former
independent-user M4 gate with a separate maintainer-controlled clean-room
consumer gate. That gate passed, so M4 is complete. This does not create an
independent user or prove adoption.

The tag/version binding, public-download result schema, workflow, and external-
attempt issue contract are implemented; the same-repository verifier is now
exercised. ADR 0015 retains Velox despite the documented command and
search collisions; those risks no longer create a replacement-name gate.

The bounded M4 internal security review is complete in
`docs/engineering/08-m4-security-review.md`. It traces build, browser/IPC, and
release flows, records one resolved Windows drive-relative runtime-path gap,
and preserves the local-tampering, unsigned-channel, and pure-Go COM residual
risks without calling the review independent.

The first maintenance-cost snapshot is complete in
`docs/product/04-maintenance-cost-record.md` and
`docs/product/maintenance-cost-v1.json`. It records the implementation window,
repository surface, recurring scheduled job ceiling, and manual preview steps
without inventing person-hours.

SignPath onboarding, Authenticode verification against real provider output,
release-mode signing records, and authenticated artifact attestations are
deferred until a real adoption trigger in ADR 0011 is observed. The existing
preparation remains fail-closed and dormant.

## M5: Product Decision

Status: Active. M4 is complete under ADR 0016. No qualifying independent-user
attempt is currently recorded.

Choose one:

- Continue toward a stable narrow packager.
- Reposition around CI packaging and reproducibility.
- Merge the useful work into an existing ecosystem.
- Stop the project.

The decision uses benchmark evidence, external user attempts, maintenance cost,
security review findings, and the strength of the PWA and Neutralino
counterarguments.

The Wails cold-build result, two accepted M3 product decisions, bounded
maintenance-cost record, internal security review, and public M4 distribution
evidence supply M5 inputs, not the product decision. M5 starts with zero
independent-user attempts; that absence is negative market evidence, not a
missing technical proof. A positive beta or broader-support decision must
either obtain qualifying adoption evidence or explicitly accept the risk in a
later ADR.
The public identity decision is complete under ADR 0015 and is not a remaining
gate. ADR 0008 records the explicit PWA and Neutralino counterarguments; user
attempts must now test whether its narrow portable-artifact boundary has real
value.

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
