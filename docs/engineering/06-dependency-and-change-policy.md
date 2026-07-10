# Dependency and Change Policy

- Status: Draft
- Owner: Project maintainer

## Goal

Dependencies must not erase the product's cold-build, cache, startup, security,
or maintainability advantage.

## Dependency Classes

### Consumer build dependencies

Must be contained in one pinned Velox release bundle. A consumer build cannot
install a compiler, package manager, SDK, or framework dependency.

### Maintainer build dependencies

May include the Go toolchain, Windows SDK, WebView2 SDK, and a C++23 toolchain
for the M0 reference host. They do not become consumer prerequisites.

### Runtime dependencies

The initial external runtime dependency is Evergreen WebView2. Additional
runtime DLLs or services require an ADR and distribution analysis.

### Development-only dependencies

Test and benchmark tools must remain outside consumer output and cannot be
required for normal application startup.

## Admission Checklist

A new dependency documents:

- Exact responsibility and owning component.
- Why a standard-library or existing dependency is insufficient.
- License and redistribution obligations.
- Maintainer and security posture.
- Artifact, startup, memory, and CI-cache effect.
- Network or installation behavior.
- Supported Windows and architecture impact.
- Removal or replacement path.
- Tests that fail if the dependency contract breaks.

## Rejection Rules

Reject a dependency that:

- Adds a consumer compiler or package-manager step.
- Requires a local server, daemon, plugin registry, or automatic network call.
- Introduces GPL or AGPL obligations into distributed core artifacts without an
  explicit legal decision.
- Hides native capability behind dynamic reflection.
- Duplicates a small stable function already owned locally.
- Cannot be pinned and checksummed in release artifacts.
- Adds generated bindings without a deterministic drift check.

## Version Policy

- Pin release and benchmark inputs exactly.
- Do not use latest in reproducible contracts.
- Record host, manifest, IPC, and build-result compatibility independently.
- Major dependency upgrades require compatibility and performance evidence.
- Security updates remain narrow unless broader migration is explicitly
  justified.

## Change Classification

| Change | Required treatment |
| --- | --- |
| Internal implementation only | Focused tests and no public claim |
| Public CLI or JSON change | Contract sync and compatibility classification |
| Manifest or IPC change | Version decision, fixtures, migration note |
| Host or WebView2 change | Windows smoke and startup comparison |
| Packaging change | Reproducibility and artifact inspection |
| New native capability | Product decision, ADR, threat model, negative tests |
| New platform | Separate architecture, support, release, and benchmark decision |

## Supply Chain

- Release downloads use published checksums.
- Third-party notices list distributed dependencies.
- Release artifacts include a software bill of materials before alpha.
- GitHub Actions references are immutable revisions when workflows are added.
- Signing credentials never enter this repository.

## Current State

No implementation dependency manifest exists yet. Exact Go, Windows SDK,
WebView2 SDK, and test-tool versions remain UNDECIDED until M0 selects the
smallest viable toolchain.
