# Release

- Status: Unsigned alpha evidence pipeline implemented; public distribution unavailable
- Owner: Project maintainer

## Current State

Velox has no published release, package registry entry, signing process, or
stable version policy. Maintainer tooling builds the Go CLI and host, assembles
the deterministic unsigned Windows x64 bundle, verifies artifact entries
against the release manifest, and emits checksums, a file-level SPDX 2.3 SBOM,
and one unsigned in-toto/SLSA provenance statement. The alpha-evidence workflow
builds the bundle twice and rejects differing ZIP bytes.

A separate consumer job performs no source checkout and invokes no Go, Node,
Rust, C++, Bun, or package-manager command. It downloads the producer artifact,
verifies its checksum, initializes and validates a project, runs doctor, builds
twice, checks deterministic ZIP hashes, and inspects the result. Hosted runner
images can still contain preinstalled toolchains; the claim is that the
consumer job does not invoke them.

## Proposed Release Unit

During MVP, the CLI, generic host, JavaScript bridge, schemas, and
compatibility metadata release atomically as one versioned bundle.

Independent component releases are deferred until a real compatibility need
exists.

## Channels

Planned channels are alpha, beta, and stable. Exact version numbers and SemVer
policy remain UNDECIDED before public alpha. Local artifacts currently identify
the development release as `0.5.5-dev`.

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

The current local bundle includes the CLI, unchanged host, strict host
metadata, product and checkout-free-consumer JSON schemas, release manifest,
and third-party notices. The release builder uses an explicit schema allowlist
and fails when a required product schema is missing. Benchmark and other CI
evidence schemas remain maintainer contracts and are not copied into the
consumer archive.

Checksums, SPDX, and provenance are workflow artifacts, not contents of the
consumer ZIP. The provenance statement is deterministic metadata but is not a
signed attestation. An attacker who can replace both release and evidence can
still forge the complete unsigned set. Authenticated provenance, signatures,
compatibility notes, and public release publication therefore remain M4 gates.

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

The current workflow does not promote or publish anything and has only
`contents: read`. Tag and manual runs produce retained workflow artifacts for
review. A future publishing workflow requires a separate approval, signing
decision, and writable GitHub permission boundary.

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
