# Operational Contract

- Status: Pre-implementation
- Primary owner: Project maintainer

## Product Shape

Velox has two local operational paths:

1. A developer or CI runner builds a portable application.
2. An application user starts the generic host and WebView2 content.

Velox has no hosted service, database, control plane, account system, or
background daemon.

## Critical Journeys

### Consumer build

A pinned Velox release and valid static project produce portable output without
a compiler, Node.js, network access, or consumer Actions cache upload.

### Application startup

A valid package starts against an installed supported WebView2 Runtime, renders
trusted local content, and either becomes ready or returns a local diagnostic.

### Artifact inspection

A developer can identify release and contract versions, permissions, file
counts, and digests without executing the artifact.

## Operational Priorities

1. Preserve source and prior output.
2. Fail closed at security and compatibility boundaries.
3. Keep builds reproducible and offline.
4. Keep diagnostics local and actionable.
5. Avoid persistent operational infrastructure.

## Service-Level Terms

Hosted-service SLO, RTO, and RPO do not apply because Velox operates no service
or authoritative remote data.

Release recovery is artifact based: preserve immutable previous releases and
allow users to select a known-good version.

## External Dependencies

- GitHub Releases or an equivalent static artifact host for distribution.
- GitHub Actions for project and benchmark CI when configured.
- Installed Evergreen WebView2 Runtime on consumer Windows systems.

No dependency is assumed operational until its workflow and failure handling
are implemented and tested.

## Release Blockers

- Consumer build requires an undeclared toolchain or network request.
- Output is not deterministic under the documented profile.
- Checksums or compatibility metadata are absent.
- Critical startup, security, or cleanup tests fail.
- Known critical risks have no owner or stop decision.
- Published performance wording exceeds reproducible evidence.

## Current Gap

The local runtime, deterministic unsigned release builder, hosted alpha
evidence workflow, guarded prerelease publisher, and no-checkout public-download
verifier exist. ADR 0012 selects `0.5.10-alpha.1` and the external-user evidence
contract. No public distribution, executed public-download result, qualifying
external-user attempt, or release operational history exists. ADR 0010's signed
channel remains deferred.
