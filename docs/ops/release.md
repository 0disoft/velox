# Release

- Status: M4 unsigned distribution complete; M5 adoption evidence remains empty
- Owner: Project maintainer

## Current State

Velox has one published unsigned developer preview and no package registry
entry, implemented signing workflow, or stable version policy. Maintainer tooling builds the Go CLI and
host, assembles the deterministic unsigned Windows x64 bundle, verifies
artifact entries against the release manifest, and emits checksums, a
file-level SPDX 2.3 SBOM, and one unsigned in-toto/SLSA provenance statement.
The alpha-evidence workflow builds the bundle twice and rejects differing ZIP
bytes.

ADR 0015 retains Velox as the maintainer-approved public identity. An existing
released Go CLI still distributes the exact `velox` command and `velox.exe`;
that collision is an accepted and disclosed release risk rather than a
replacement-name gate.

The repository also owns `velox.signing-record/v1` and a non-publishable
dry-run verifier. It binds unsigned inputs, the signing-input ZIP, signed-output
placeholders, the final manifest and ZIP, checksums, and SBOM without contacting
a provider or claiming Authenticode or artifact-attestation success.

The public repository now declares `MIT OR Apache-2.0`, identifies the
maintainer in CODEOWNERS, and includes security and privacy policies. The
SignPath application packet and exact proposed provider configuration live in
`docs/ops/signpath-onboarding.md` and `.signpath/`, but ADR 0011 defers provider
onboarding until a real adoption trigger exists.

A separate consumer job performs no source checkout and invokes no Go, Node,
Rust, C++, Bun, or package-manager command. It downloads the producer artifact,
verifies its checksum, initializes and validates a project, runs doctor, builds
twice, checks deterministic ZIP hashes, and inspects the result. Hosted runner
images can still contain preinstalled toolchains; the claim is that the
consumer job does not invoke them.

[Alpha evidence run 29714104653](https://github.com/0disoft/velox/actions/runs/29714104653)
completed the reproducible producer and checkout-free consumer jobs for tag
`v0.5.10-alpha.1` at commit
`9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Manual publication
[run 29714173324](https://github.com/0disoft/velox/actions/runs/29714173324)
created the immutable prerelease. Public-download verification
[run 29715002921](https://github.com/0disoft/velox/actions/runs/29715002921)
downloaded the release without source checkout, matched ZIP SHA-256
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`,
verified the sidecars, built twice, inspected, and reached startup-ready. This
is same-repository public-download evidence, not an independent external-user
attempt or authenticated attestation.

The now-archived public
[`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository consumed only the pinned public release. Hosted
[run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250)
passed the full public CLI path with no consumer toolchain command and zero
Actions cache upload bytes. ADR 0016 accepts this maintainer-controlled clean-
room result as the final technical M4 gate while preserving
`maintainerControlled: true` and `externalUserAttempt: false`.
It remains read-only as a one-shot receipt. Future release verification uses
the repository-owned public-preview workflow instead of advancing that pin.

## Proposed Release Unit

During MVP, the CLI, generic host, JavaScript bridge, schemas, and
compatibility metadata release atomically as one versioned bundle.

Independent component releases are deferred until a real compatibility need
exists.

## Channels

Planned channels are alpha, beta, and stable. `0.5.10-alpha.1` is the published
unsigned developer preview at immutable tag `v0.5.10-alpha.1`. Public artifacts and executables use the Velox identity
fixed by ADR 0015. ADR 0018 defines the clean-room LLM agent evidence required
before beta technical readiness. Actual beta or stable promotion, support
policy, signing, and publication remain separate maintainer decisions after
the qualifying trial set exists.

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
- Unsigned provenance metadata before the developer preview.
- Prominent unsigned, SmartScreen, and managed-device limitations.

The current local bundle includes the CLI, unchanged host, strict host
metadata, product and checkout-free-consumer JSON schemas, release manifest,
and third-party notices. The release builder uses an explicit schema allowlist
and fails when a required product schema is missing. Benchmark and other CI
evidence schemas remain maintainer contracts and are not copied into the
consumer archive.

Checksums, SPDX, and provenance are release assets, not contents of the
consumer ZIP. The provenance statement is deterministic metadata but is not a
signed attestation. An attacker who can replace both release and evidence can
still forge the complete unsigned set. ADR 0011 accepts that boundary for a
developer preview and requires it to be disclosed. ADR 0010 retains separate
authenticated provenance and Authenticode controls for a later signed channel.

## Release Gates

- The Velox identity and known command/search collisions are disclosed and
  accepted under ADR 0015.
- All configured correctness and Windows smoke checks pass.
- Unsigned reproducibility passes where applicable.
- Consumer build requires no compiler, Node.js, or Actions cache.
- Security baseline tests pass.
- Performance wording is regenerated from current benchmark evidence.
- Critical risks are mitigated, accepted explicitly, or stop the release.
- Directory asset tampering, branding, signing, and platform limitations are
  visible.
- The preview is marked prerelease and prominently identifies both executables
  as unsigned.
- Publication requires a manual exact-phrase confirmation on an existing alpha
  tag and refuses to replace an existing release.

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

ADR 0011 owns the unsigned developer-preview boundary. No Authenticode provider
or signing credential is required for that channel. ADR 0010 and
`docs/ops/signing.md` own a future signed-channel boundary. SignPath Foundation
remains a conditional provider candidate; Microsoft Artifact Signing remains a
migration candidate where eligibility and publisher identity fit.

The provider signs the reproducibly built `velox.exe` and `velox-host.exe`.
The repository-owned `velox-signing-record prepare` command packages exactly
those two unsigned files into a deterministic, self-verified signing input
without contacting the provider.
The separate `authenticode` command then fails closed unless the returned
directory contains exactly those two names, both signatures are valid, both
use the approved exact publisher subject and SHA-256, both have timestamp
certificate identities, and both share one signer certificate.
The final bundle is then assembled from those exact signed inputs so
`velox-host.json` and `release-manifest.json` describe signed bytes. The generic
host remains byte-identical after release and during application packaging, so
its signature is preserved. Application-specific executable branding and
signing are not part of the initial release.

Signing credentials stay outside this repository. No private key or PFX enters
GitHub secret storage. Provider submission credentials, approval, and release
write permission belong to separate protected-environment gates.

## Developer-Preview Publication

ADR 0015 removes the replacement-name gate. This mechanism remains manual and
must not run until the candidate is rebuilt and every evidence gate below
passes.

Ordinary pull-request, tag, and evidence runs retain workflow artifacts and
have only `contents: read`. A manual dispatch can publish only when
`publish_preview` is true, the exact confirmation phrase is supplied, and the
selected ref is an existing `vX.Y.Z-alpha.N` tag. The isolated publication job
alone receives `contents: write`.

That job downloads the producer evidence after the checkout-free consumer job
passes, rejects missing or extra files, verifies every checksum, refuses an
existing release, and creates an immutable GitHub prerelease with the unsigned
warning. It also rejects a tag that is not exactly `v<releaseVersion>`. It does
not sign, attest, rebuild, or replace artifacts. The immutable release notes
also state the Windows and WebView2 compatibility floor, directory-asset
tampering boundary, unchanged-host branding limitations, missing installer and
updater, and the accepted `velox` command and executable-name collision.

Promotion to a future signed, beta, or stable channel reuses an already
verified immutable candidate. It does not relabel unsigned bytes as signed or
rebuild different bytes under the same version.

## Stop Conditions

- Reproducibility or checksum verification fails.
- Release artifact behavior differs from tested artifacts.
- Required WebView2 support cannot be stated accurately.
- Benchmark results fail the roadmap go-or-kill gate.
- Product or executable identity differs from ADR 0015.
- Security reporting and release ownership are not ready for public use.

## Post-Release Verification

For the first preview, a separate maintainer-controlled repository verified
download, checksum, version inspection, hello build, and application startup
from the public asset. Run `29736140250` completed that one-shot post-release
gate. The repository is now archived; this is not a recurring release design.

The repository-owned `Public preview verification` workflow covers the public
URL, independently supplied digest, checksum, SPDX, provenance, tag/version,
build, inspection, and startup boundaries without source checkout. Its result
is explicitly same-repository evidence. Neither it nor the clean-room consumer
repository substitutes for qualifying adoption evidence defined in
`docs/ops/external-user-attempt.md`.

For the first preview, the workflow initially exhausted its eight-minute job
limit because the ordinary generated starter did not emit the benchmark-only
ready marker. Commit `17a91f5c90dcbd58cf8aa20836994097e9c3262b`
made the verifier inject that marker only into its temporary fixture and bound
each downloaded CLI invocation to 120 seconds. The successful rerun is the
current post-release verification record.
