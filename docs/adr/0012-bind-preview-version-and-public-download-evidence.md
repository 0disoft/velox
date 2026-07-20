# ADR 0012: Bind preview version and public-download evidence

- Status: Accepted; external-user M4 gate superseded by ADR 0016
- Date: 2026-07-18
- Owner: Project maintainer

## Context

ADR 0011 permits an unsigned developer preview, but publication still needs a
precise artifact identity. A tag can otherwise name bytes whose embedded CLI,
host metadata, and release manifest report another version. Same-workflow
artifact consumption also cannot prove that the immutable GitHub Release URL
serves the intended files.

The current development line is `0.5.10-dev`. No public version has been
published, so the first candidate can adopt a prerelease version without a
compatibility migration.

## Decision

Use `0.5.10-alpha.1` as the first public developer-preview candidate and
`v0.5.10-alpha.1` as its only valid tag.

Every preview publication must enforce these rules:

1. The selected tag is exactly `v<releaseVersion>` from
   `release-manifest.json`.
2. The CLI, generic host metadata, release manifest, SBOM, and provenance use
   the same release version.
3. Publication reuses the producer artifact after checkout-free consumer
   verification; it does not rebuild.
4. A separate workflow downloads the four release assets from the public
   GitHub Release URL without source checkout.
5. Public-download verification requires a SHA-256 supplied independently of
   the downloaded checksum file, then verifies the checksum, SPDX, provenance,
   manifest, CLI version, deterministic consumer build, and startup-ready path.
6. Same-repository public-download evidence records
   `externalUserAttempt: false`. It cannot complete the external-user M4 gate.
7. A real external attempt must come from an account or repository not
   controlled by the implementation workflow and must identify the exact tag
   and digest without exposing private paths or credentials.

Development versions may use `X.Y.Z-dev`. Public alpha versions use
`X.Y.Z-alpha.N`, where `N` starts at 1 and increases for every published byte
change. Published tags and assets are immutable.

## Alternatives

### Keep `0.5.10-dev` inside the first tagged artifact

Rejected. A public tag and embedded development identity would make support,
reproduction, and rollback ambiguous.

### Trust only checksums downloaded beside the ZIP

Rejected. Co-located checksums detect accidental corruption but do not provide
an independent expected digest when the entire release channel is replaced.

### Treat the repository's verification workflow as an external user

Rejected. It proves the public acquisition path, not independent adoption or
documentation usability.

### Rebuild during public-download verification

Rejected. Verification must exercise the published bytes, not create another
candidate.

## Consequences

### Positive

- Tags, manifests, executables, and evidence have one support identity.
- Public hosting and release-asset drift are tested separately from build-job
  artifact transfer.
- The first external report has a stable digest and environment record.
- M4 cannot be closed by a same-owner workflow marking its own work successful.

### Negative

- The candidate version must change before any further public alpha bytes are
  published.
- Public-download verification depends on GitHub Releases and runner network
  availability.
- The independently supplied digest still needs a trusted handoff from the
  release workflow or maintainer.
- A human external-user report remains necessary after automation passes.

## Validation

The release bundle test requires the public-verification schema in the consumer
archive. Workflow hygiene tests require immutable action pins, no checkout or
language toolchain in the public verifier, exact tag/version binding, public URL
acquisition, independent SHA-256 input, startup, and the explicit non-external
evidence marker.

The public workflow cannot pass before a release exists. Its first successful
run ID and result artifact become M4 evidence only after the release is
published.

## Rollback or Fallback

Before tagging, replace the candidate version through an ordinary reviewed
commit. After publication, never move or replace the tag or assets. Publish a
new `alpha.N` version and mark the bad preview unsuitable.

If GitHub Releases is unavailable, preserve the verified candidate and wait;
do not substitute a same-run artifact while claiming public-download evidence.

## Revisit Triggers

- A second alpha is required.
- M5 selects beta or stable continuation.
- Distribution moves away from GitHub Releases.
- Authenticated attestations or Authenticode become active.
- The public verification result needs an independently signed digest source.

## Synchronized Surfaces

- `README.md`
- `VALIDATION.md`
- `docs/README.md`
- `docs/adr/README.md`
- `docs/engineering/06-dependency-and-change-policy.md`
- `docs/ops/00-operational-contract.md`
- `docs/ops/external-user-attempt.md`
- `docs/ops/release.md`
- `docs/product/01-roadmap.md`
- `.github/workflows/alpha-evidence.yml`
- `.github/workflows/public-preview-verification.yml`
- `.github/ISSUE_TEMPLATE/external-user-attempt.yml`
- `schema/public-preview-verification-v1.schema.json`
- release-bundle and workflow hygiene tests
