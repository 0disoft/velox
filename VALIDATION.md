# Validation

- Status: M4 complete; M5 decision accepted for narrow alpha continuation; beta remains gated with zero independent-user attempts

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
- `velox_alpha_evidence_smoke` verifies the release manifest and emits local
  checksum, SPDX, and unsigned provenance evidence for that bundle.
- `velox_signing_record_smoke` runs the deterministic signing-input packager,
  repository-owned signing-record package, and maintainer CLI tests; emits a
  non-publishable dry-run record; validates it against
  `velox.signing-record/v1`; and proves `publishable: true` is rejected for
  dry-run evidence. The Go test suite also exercises the fail-closed
  Authenticode policy boundary and `velox.authenticode-verification/v1`; a real
  signed-provider success remains a deferred future-channel gate rather than an
  M4 requirement.
- `velox_signpath_onboarding_smoke` verifies the repository-owned SignPath
  artifact configuration, GitHub source policy, dual-license files,
  CODEOWNERS, security policy, privacy policy, and application handoff packet.
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
- `velox_deskboard_model_test` exercises the functional example's persisted
  task-state normalization, mutations, filters, and derived progress without a
  browser or frontend dependency.
- `velox_deskboard_smoke` validates, diagnoses, builds twice, compares archive
  hashes, inspects, starts the packaged application directly from a non-app
  working directory, and starts `examples/deskboard` through the assembled
  Velox release. The harness is Bun/TypeScript and adds no PowerShell surface.
- `velox_deskboard_build` leaves a portable Deskboard directory and ZIP under
  `dist/examples/deskboard` for manual use.
- `velox_capability_probe_smoke` validates, diagnoses, reproducibly builds,
  inspects, directly starts, and source-starts the browser capability probe.
- `velox_capability_probe_model_test` verifies operation-result replacement,
  rerun preservation, evidence-state summaries, and versioned report snapshots.
- `velox_capability_probe_build` leaves a portable probe directory and ZIP
  under `dist/examples/capability-probe` for manual user-gesture checks.
- `velox_example_tooling_test` verifies that the maintainer example builder can
  replace only the allowlisted `dist/examples` outputs and rejects
  arbitrary output names.
- `velox_file_notes_model_test` verifies draft restoration, dirty-state
  derivation, selected-file baselines, saved baselines, and Unicode statistics.
- `velox_file_notes_smoke` validates, diagnoses, reproducibly builds, inspects,
  directly starts, and source-starts the browser-owned file editor.
- `velox_file_notes_build` leaves a portable File Notes directory and ZIP under
  `dist/examples/file-notes` for manual picker and persistence checks.

The hosted `Alpha release evidence` workflow builds the unsigned release twice,
requires byte-identical ZIPs, generates checksum, SPDX, and unsigned provenance
artifacts, and passes the artifact to a checkout-free consumer job. That job
invokes only `velox.exe`; it does not prove signing, authenticated provenance,
public-release download, or adoption by an external user.

An explicit manual dispatch can publish those verified files only from an
existing `vX.Y.Z-alpha.N` tag after the exact unsigned-preview confirmation is
entered. The isolated publication job alone receives `contents: write`, refuses
replacement, and creates a prerelease with SmartScreen and managed-device
warnings. Workflow validation proves this contract; it does not publish a
release.

Manual hosted [run 29806946109](https://github.com/0disoft/velox/actions/runs/29806946109)
passed for exact commit `d8495b8aa2a399505b583a8ed881b5bc7fa9f304` after the
browser-owned file workflow examples were added. The reproducible release and
checkout-free consumer jobs succeeded; publication was disabled and skipped.
ADR 0017 treats this as technical alpha evidence, not independent adoption or
permission to add an application-specific Go backend or broad native API.

ADR 0015 retains Velox as the maintainer-approved public identity and supersedes
ADR 0013's replacement-name gate. The known `velox.exe` and search collisions
remain documented risks, but the publication job may run for the exact
`0disoft/velox` repository after every ordinary unsigned-preview gate passes.

The manual `Public preview verification` workflow performs no source checkout
and downloads the ZIP, checksum, SPDX, and provenance assets from the public
GitHub Release URL. It requires an independently supplied ZIP SHA-256, binds the
tag to the release manifest and CLI version, builds twice, inspects, and reaches
the startup-ready marker. Its schema fixes `externalUserAttempt` to `false`, so
this same-repository check cannot prove independent adoption.

The first public preview is
[`v0.5.10-alpha.1`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.1)
from commit `9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Tag evidence run
`29714104653`, publication run `29714173324`, and public-download verification
run `29715002921` passed. The verifier downloaded SHA-256
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`,
validated the sidecar evidence, built twice, inspected the output, and reached
startup-ready without source checkout. This is current release evidence, not an
external-user attempt or authenticated publisher identity.

The now-archived separate public
[`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository consumed the pinned release without checking out Velox source.
Hosted [run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250)
at consumer commit `ed003602d65cbaef12bf95ee78b2cf16466bdfcd`
validated every release sidecar, all seven public CLI paths, deterministic
build output, inspection, and startup. The evidence records no consumer
toolchain command and zero Actions cache upload bytes. ADR 0016 accepts this as
the technical M4 distribution gate while requiring
`maintainerControlled: true` and `externalUserAttempt: false`; it is not
independent adoption evidence. The repository is retained read-only as the
one-shot receipt; future release verification stays in this repository.

The bounded M5 readiness records are `docs/product/maintenance-cost-v1.json`,
`docs/product/04-maintenance-cost-record.md`, and
`docs/engineering/08-m4-security-review.md`. Hygiene tests validate their
version, observation boundary, non-claim language, roadmap synchronization,
and the unsigned-preview security baseline. The security review remains
internal and does not replace external-user evidence.

The hosted `Consumer evidence` workflow additionally runs three startup
lifecycle samples for pull requests and `quick` manual dispatches. A `full`
manual dispatch, weekly schedule, or release-candidate tag runs ten lifecycle
and ten consumer samples. It validates
`velox.startup-lifecycle/v3`, derives and validates
`velox.startup-lifecycle-summary/v1` plus
`velox.startup-lifecycle-phase-summary/v1`, and uploads all results with
`always()`. The phase summary computes interval p50 and p95 values and the
dominant immediate-startup interval directly from raw v3 evidence.
Lifecycle v3 preserves the host-local startup and shutdown phase timelines for
both the first launch and the immediate same-profile relaunch.
This longer evidence path is intentionally separate from the local one-sample
`velox_startup_smoke` intent.

An explicit manual `include_profile_comparison` input runs three alternating,
serial same-profile versus fresh-profile pairs and validates
`velox.startup-profile-comparison/v1`. It is disabled for ordinary pull request,
scheduled, and release-candidate evidence.

Each weekly schedule also builds `velox.startup-history/v1` from the current
lifecycle summary and up to eleven prior successful scheduled artifacts. The
history is grouped by runner image version and WebView2 version, retained for
90 days, and remains diagnostic evidence rather than an automatic regression
gate. Manual runs can exercise the same collector with
`include_startup_history`.

The `Actions warning monitor` workflow allocates a runner after scheduled or
release-candidate consumer evidence, or for an explicit manual run ID. Pull
request and ordinary manual consumer evidence produce only a skipped monitor
job. The monitor scans the bounded workflow-log archive for the known
`actions/download-artifact` `DEP0005 Buffer()` warning. It validates and uploads
`velox.actions-warning-monitor/v1`. Presence is diagnostic rather than a failed
product check; malformed or inaccessible log evidence still fails the monitor.
The platform-independent scanner uses the pinned `ubuntu-24.04` runner.

The C++23/Pixi M0 reference intents were retired after ADR 0005 selected Go
for both production executables. Historical comparison results remain in ADR
0004 and the performance budget.

Unconfigured validation names remain skipped and must not pass with a fake
success.

## M2 Security Evidence

| Contract | Executable evidence |
| --- | --- |
| Trusted virtual origin and top-level messages | `internal/webview2` origin tests and Windows startup security fixture |
| Navigation, frame, popup, download, and permission denial | Windows startup security fixture policy audit |
| Closed method and permission table | `internal/ipc` dispatcher tests |
| Payload, nesting, request ID, duplicate, and in-flight limits | `internal/ipc` malformed and concurrency tests |
| Frozen JavaScript bridge | embedded bridge contract test and Windows startup IPC invocation |
| Production development-tool restrictions | runtime security source guard and startup production path |
| No listening socket or broad native API | production-host source guard and closed dispatcher tests |
| Missing runtime and malformed configuration | startup and runtime-configuration failure tests |
| Path, archive, staging, and release checksum controls | asset, build-plan, builder, inspector, host metadata, and release tests |

The security fixture must complete both a trusted `app.getInfo` invocation and
the five browser-policy denials before emitting `security-ok`.

## Hygiene Validation

Repository hygiene file changes must check line-ending churn, binary diff pollution,
tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

general validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Repository Shape

cli-tool validation must stay repository-shape focused and must not imply generated application source code.
