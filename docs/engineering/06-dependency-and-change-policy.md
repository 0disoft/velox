# Dependency and Change Policy

- Status: Draft
- Owner: Project maintainer

## Goal

Dependencies must not erase the product's cold-build, cache, startup, security,
or maintainability advantage.

## Dependency Classes

### Consumer build dependencies

Must be contained in one pinned Actutum release bundle. A consumer build cannot
install a compiler, package manager, SDK, or framework dependency.

### Maintainer build dependencies

Includes the Go toolchain used to build the CLI and host. Windows and WebView2
runtime facilities remain platform dependencies, not consumer build
toolchains.

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
- Development builds use `X.Y.Z-dev`; public alpha artifacts use
  `X.Y.Z-alpha.N` and an exact `v<releaseVersion>` immutable tag.
- The first selected public candidate is `0.6.0-alpha.1`; any changed bytes
  after publication require a higher alpha sequence.
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

M0 pins `github.com/jchv/go-webview2` to commit `56598839c808` through pseudo
version `v0.0.0-20260205173254-56598839c808`. It is MIT licensed, pure Go, and
loads its embedded WebView2 loader through `go-winloader`. Wails also uses this
binding on Windows, so the dependency itself is not a performance moat.

The binding is acceptable for a startup feasibility spike but not yet for the
product host. Its public API does not expose all virtual-origin and browser
policy controls required by the security baseline, and its constructor enables
clipboard-read permission. Removal cost is limited because all usage is
confined to `cmd/actutum-host` during M0.

The retired C++23 reference environment used Pixi, Clang, CMake, lld, Ninja,
the WebView2 SDK, Visual Studio headers, and the Windows SDK. Those dependencies
are preserved only as historical evidence in ADR 0004 and are no longer part
of the active repository graph.
