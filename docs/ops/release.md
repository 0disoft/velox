# Release

- Status: Pre-implementation
- Owner: Project maintainer

## Current State

Velox has no implementation release, package registry entry, signing process,
or stable version policy. Design documents are not a software release.

## Proposed Release Unit

During MVP, the CLI, generic host, JavaScript bridge, schemas, and
compatibility metadata release atomically as one versioned bundle.

Independent component releases are deferred until a real compatibility need
exists.

## Channels

Planned channels are alpha, beta, and stable. Exact version numbers and SemVer
policy remain UNDECIDED until M1 creates public contracts.

Nightly distribution is not planned during the initial project stage.

## Required Release Contents

- Windows x64 Velox bundle.
- CLI and unchanged generic host.
- JavaScript bridge and schemas.
- Release manifest with contract versions and artifact digests.
- SHA-256 checksums.
- Software bill of materials.
- Third-party notices.
- Compatibility and known-limitation notes.
- Provenance before public alpha.

## Release Gates

- All configured correctness and Windows smoke checks pass.
- Unsigned reproducibility passes where applicable.
- Consumer build requires no compiler, Node.js, or Actions cache.
- Security baseline tests pass.
- Performance wording is regenerated from current benchmark evidence.
- Critical risks are mitigated, accepted explicitly, or stop the release.
- Directory asset tampering, branding, signing, and platform limitations are
  visible.

## Signing Boundary

The generic host remains byte-identical when packaged, so its vendor signature
can remain valid. Application-specific executable branding and signing are not
part of the initial release.

Signing credentials stay outside this repository. The exact signing provider
and promotion workflow remain UNDECIDED.

## Promotion

Promotion reuses an already verified immutable artifact. It does not rebuild
different bytes for stable.

## Stop Conditions

- Reproducibility or checksum verification fails.
- Release artifact behavior differs from tested artifacts.
- Required WebView2 support cannot be stated accurately.
- Benchmark results fail the roadmap go-or-kill gate.
- Security reporting and release ownership are not ready for public use.

## Post-Release Verification

When release automation exists, it must verify download, checksum, version
inspection, hello build, and application startup from the published artifact.

No command is documented here until that command is configured and exercised.
