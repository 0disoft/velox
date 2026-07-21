# Velox

- Status: M4 complete; M5 product decision active with no independent adoption recorded
- Scope: general
- Repository type: cli-tool

Velox is a compile-free Windows desktop application packager. It is designed
to turn static HTML, CSS, and JavaScript into a
portable WebView2 application without compiling application-specific native
code.

The repository now contains a policy-enforcing pure-Go WebView2 host, a frozen
and permission-checked IPC v1 bridge, manifest
validation, an immutable build plan, atomic portable-directory assembly, a
deterministic ZIP writer, all seven M1 CLI commands, an unsigned deterministic
Windows x64 release bundle, startup fixtures, zero-cache consumer evidence,
and an alpha evidence workflow. The workflow builds the release twice, emits
checksums, a file-level SPDX SBOM, an unsigned in-toto/SLSA provenance
statement, and then exercises it from a checkout-free consumer job. A guarded
manual job publishes those exact files as an explicitly unsigned developer
preview.

[Velox v0.5.10-alpha.1](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.1)
is the first public unsigned developer preview. Tag evidence
[run 29714104653](https://github.com/0disoft/velox/actions/runs/29714104653)
and publication [run 29714173324](https://github.com/0disoft/velox/actions/runs/29714173324)
produced the immutable release from commit
`9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Public-download verification
[run 29715002921](https://github.com/0disoft/velox/actions/runs/29715002921)
then exercised the release without source checkout at verifier commit
`17a91f5c90dcbd58cf8aa20836994097e9c3262b`. The ZIP SHA-256 is
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`.
The executables remain unsigned and the provenance remains unauthenticated
metadata.

## Headline Metrics

1. End-to-end cold build time.
2. Consumer GitHub Actions cache upload.

Process-to-ready startup is a release guardrail and lifecycle diagnostic, not a
headline advantage. Velox intentionally trades native feature breadth for a
smaller build and runtime surface.

## Proposed Shape

- A standalone Go CLI validates and packages projects.
- A separate prebuilt generic host opens static assets through WebView2.
- The production host is pure Go with no CGo or C++ shim.
- The retired C++23 M0 comparison remains available as historical ADR and
  performance evidence, not as an active build target.
- Consumer builds copy an unchanged host, external configuration, and assets.
- Consumer builds require no compiler, Node.js, or frontend package manager.

## Current Product Boundary

Supported by the MVP design:

- Windows x64.
- Static web assets.
- One top-level window.
- Portable directory and deterministic ZIP output.
- Minimal versioned JSON IPC for application information and basic window
  lifecycle.
- Non-interactive CLI operation and machine-readable output.

Explicitly deferred:

- Native application backends and plugins.
- Filesystem, shell, process, and sidecar APIs.
- Frontend bundling, hot reload, and a development server.
- Installers, automatic updates, per-application executable branding, and code
  signing automation.
- macOS, Linux, ARM64, and multi-window support.

## Documentation

- Product scope: docs/product/02-spec.md
- Product brief: docs/product/00-product-brief.md
- Roadmap: docs/product/01-roadmap.md
- Risk register: docs/product/03-risk-register.md
- Architecture: ARCHITECTURE.md
- Initial decision: docs/adr/0001-initial-architecture-boundaries.md
- CLI contract: docs/cli/command-contract.md
- Performance budget: docs/engineering/03-performance-budget.md
- Security policy: SECURITY.md
- Privacy policy: PRIVACY.md
- Unsigned preview decision: docs/adr/0011-publish-unsigned-developer-preview-before-signing.md
- Preview identity decision: docs/adr/0012-bind-preview-version-and-public-download-evidence.md
- Public-name decision: docs/adr/0015-retain-velox-public-identity.md
- Distribution/adoption boundary: docs/adr/0016-separate-technical-distribution-from-independent-adoption.md
- Deferred SignPath onboarding: docs/ops/signpath-onboarding.md
- External user attempt: docs/ops/external-user-attempt.md

## Current CLI Slice

The CLI expects an unchanged prebuilt `velox-host.exe` and strict
`velox-host.json` beside `velox.exe` in a release bundle. It verifies release,
target, host-contract, runtime-contract, IPC-contract, file-size, and SHA-256 agreement before
building. Consumer builds never invoke Go, C++, Node.js, Pixi, or a package
manager.

```powershell
velox init .\hello --json
velox validate --config .\velox.json --json
velox doctor --config .\velox.json --out .\dist --json
velox run --config .\velox.json --out .\.velox-run --json
velox build --config .\velox.json --out .\dist --json
velox inspect .\dist\dev.velox.hello.zip --json
velox version --json
```

`build` produces `dist/<app-id>/`, `dist/<app-id>.zip`, and a deterministic
`build-result.json` inside the portable directory and archive. The host bytes
are copied unchanged. Output assembly occurs in an owned sibling staging path;
an occupied staging or recovery path fails closed instead of deleting it.

See `examples/hello/velox.json` and `schema/velox-v1.schema.json` for the v1
authoring contract.

## Functional Test App

`examples/deskboard` is a complete local-first task board built from static
HTML, CSS, and JavaScript. It persists versioned task data in the application
WebView2 profile, exercises the supported app and window IPC methods, and ships
without a frontend dependency or bundler. Its model tests and packaging/startup
smoke prove a more realistic application path than the minimal hello fixture.

`examples/capability-probe` reports which browser-owned storage, file-picker,
clipboard, drag-and-drop, and permission surfaces are exposed by the current
WebView2 environment. It keeps every operation user-initiated and does not add
or imply a Velox native capability.

`examples/file-notes` uses those browser-owned surfaces in a file-backed
Markdown editor with explicit open and save gestures, IndexedDB draft recovery,
unsaved-change protection, and no Velox native permission.

## Development State

M0 selected the pure-Go WebView2 host, M1 completed the compile-free packaging
slice, and M2 closed the minimum runtime security contract. M3 has passed its
publishable Wails cold-build gate and its narrowly defined structural-
simplicity gate. Startup has been removed from the headline and retained as a
release guardrail. M3's public benchmark deliverables and hosted evidence are
complete. M4 has local and hosted unsigned alpha evidence, a published public
developer preview, same-repository public-download verification, and a
separate public clean-room consumer repository.
Deterministic signing-input, lineage, and
fail-closed Authenticode verification tooling remain dormant for a future
signed channel. ADR 0016 closes M4 on technical distribution evidence and
starts M5 with zero independent-user attempts recorded; provider-approved
signing and authenticated provenance are not M4 gates. The published preview
is `0.5.10-alpha.1`. Neither same-repository verification nor the
maintainer-controlled consumer repository counts as independent adoption.

The now-archived public
[`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository consumed only the pinned public release once in hosted
[run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250).
It invoked no consumer compiler, Node.js, package manager, or Actions cache and
passed release, deterministic-build, inspection, and startup checks. Its schema
fixes `maintainerControlled: true` and `externalUserAttempt: false`.
The repository is retained read-only as historical evidence; ongoing public-
release verification remains in Velox itself.

The bounded maintenance-cost snapshot and internal M4 security review are now
complete M5 inputs. They explicitly record the maintained WebView2 fork,
weekly hosted-job ceiling, unsigned-channel trust limit, and accepted mutable-
asset boundary. They are not person-hour estimates or an independent audit.

ADR 0015 retains Velox as the maintainer-approved product, command, module,
schema, and release identity. The collision review still records Meta's
established project and an existing Go CLI that ships the exact `velox` command
and `velox.exe`; those are accepted discovery and command risks rather than a
replacement-name publication gate.

Consumer release packaging is published as an unsigned developer preview.
`init` creates a
dependency-free starter, `doctor` checks the current Windows, WebView2, project,
and bundled-host compatibility, `run` launches source assets through the
prebuilt host without a development server, and `inspect` validates both
portable directories and ZIPs
without executing them. The parent workspace exposes bounded
maintainer-only release-bundle, compiler-free consumer smoke, host smoke, and
benchmark intents documented in `DEVELOPMENT.md` and `VALIDATION.md`.
The repository-owned consumer workflow keeps maintainer compilation in a
producer job, measures isolated consumer jobs from artifact acquisition through
portable ZIP inspection, and publishes raw and aggregated result contracts.

## Repository Workflow

- Agent instructions: AGENTS.md
- Validation names: VALIDATION.md
- Checklist router: CHECKLIST.md
- Documentation index: docs/README.md
- Scaffold state: .ssealed/manifest.json

## License

Velox is licensed under `MIT OR Apache-2.0`, at your option. See `LICENSE-MIT`
and `LICENSE-APACHE`. Third-party attributions are listed in
`THIRD_PARTY_NOTICES.md`.
