# Release

- Status: Signing design accepted; provider onboarding and public distribution unavailable
- Owner: Project maintainer

## Current State

Velox has no published release, package registry entry, implemented signing
workflow, or stable version policy. Maintainer tooling builds the Go CLI and
host, assembles the deterministic unsigned Windows x64 bundle, verifies
artifact entries against the release manifest, and emits checksums, a
file-level SPDX 2.3 SBOM, and one unsigned in-toto/SLSA provenance statement.
The alpha-evidence workflow builds the bundle twice and rejects differing ZIP
bytes.

The repository also owns `velox.signing-record/v1` and a non-publishable
dry-run verifier. It binds unsigned inputs, the signing-input ZIP, signed-output
placeholders, the final manifest and ZIP, checksums, and SBOM without contacting
a provider or claiming Authenticode or artifact-attestation success.

A separate consumer job performs no source checkout and invokes no Go, Node,
Rust, C++, Bun, or package-manager command. It downloads the producer artifact,
verifies its checksum, initializes and validates a project, runs doctor, builds
twice, checks deterministic ZIP hashes, and inspects the result. Hosted runner
images can still contain preinstalled toolchains; the claim is that the
consumer job does not invoke them.

[Alpha evidence run 29631165931](https://github.com/0disoft/velox/actions/runs/29631165931)
completed both jobs at commit
`744b7809a0f82cad66a2936702abd4518287a551`. A separate artifact download
verified all three checksum entries, the SPDX 2.3 and in-toto/SLSA document
identities, the bundled consumer-result schema, and identical first and second
consumer build hashes. This remains same-workflow evidence, not an independent
public-download verification.

## Proposed Release Unit

During MVP, the CLI, generic host, JavaScript bridge, schemas, and
compatibility metadata release atomically as one versioned bundle.

Independent component releases are deferred until a real compatibility need
exists.

## Channels

Planned channels are alpha, beta, and stable. Exact version numbers and SemVer
policy remain UNDECIDED before public alpha. Local artifacts currently identify
the development release as `0.5.7-dev`.

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
still forge the complete unsigned set. ADR 0010 selects separate authenticated
provenance and Authenticode controls, but their implementation, compatibility
notes, and public release publication remain M4 gates.

## Release Gates

- All configured correctness and Windows smoke checks pass.
- Unsigned reproducibility passes where applicable.
- Consumer build requires no compiler, Node.js, or Actions cache.
- Security baseline tests pass.
- Performance wording is regenerated from current benchmark evidence.
- Critical risks are mitigated, accepted explicitly, or stop the release.
- Directory asset tampering, branding, signing, and platform limitations are
  visible.

## Compatibility Floor

The alpha contract supports Windows 10 version 1709 x64 and newer client
builds, or Windows Server 2016 x64 and newer server builds. Evergreen WebView2
Runtime `92.0.902.49` is the minimum because Velox requires
`ICoreWebView2_4` to cancel downloads as part of its security baseline. Doctor
checks both floors; ordinary Evergreen updates remain supported and are the
recommended runtime path.

The floor is derived from the
[Go Windows minimum](https://go.dev/wiki/MinimumRequirements), the
[WebView2 supported Windows list](https://learn.microsoft.com/en-us/microsoft-edge/webview2/),
and Microsoft's archived WebView2 SDK release notes that bind
`ICoreWebView2_4` SDK `1.0.902.49` to Runtime `92.0.902.49`.

## Signing Boundary

ADR 0010 and `docs/ops/signing.md` own this boundary. SignPath Foundation is the
conditional Authenticode provider for public alpha; GitHub artifact attestations
authenticate the final release ZIP and SBOM. Microsoft Artifact Signing remains
the migration candidate for a project-owned publisher identity or paid service
operation.

The provider signs the reproducibly built `velox.exe` and `velox-host.exe`.
The repository-owned `velox-signing-record prepare` command packages exactly
those two unsigned files into a deterministic, self-verified signing input
without contacting the provider.
The final bundle is then assembled from those exact signed inputs so
`velox-host.json` and `release-manifest.json` describe signed bytes. The generic
host remains byte-identical after release and during application packaging, so
its signature is preserved. Application-specific executable branding and
signing are not part of the initial release.

Signing credentials stay outside this repository. No private key or PFX enters
GitHub secret storage. Provider submission credentials, approval, and release
write permission belong to separate protected-environment gates.

## Promotion

Promotion reuses an already verified immutable signed artifact. It does not
rebuild or re-sign different bytes for stable.

The current workflow does not promote or publish anything and has only
`contents: read`. Tag and manual runs produce retained workflow artifacts for
review. A future publishing workflow requires successful provider onboarding,
a deterministic signing-input packager, dry-run lineage verifier,
protected-environment approval, final artifact
attestations, and a narrowly isolated `contents: write` publication job.

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
