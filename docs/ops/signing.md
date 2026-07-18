# Signing and Attestation

- Status: Design accepted; provider onboarding and workflow implementation pending
- Owner: Project maintainer
- Decision: ADR 0010

## Purpose

This document owns the M4 trust boundary between reproducible build evidence,
Authenticode signing, final release assembly, authenticated provenance, and
promotion. It does not authorize a release or define credentials.

## Trust Claims

| Claim | Mechanism | Subject |
| --- | --- | --- |
| Two clean builds produced the same native inputs | Repository-owned equality check | Unsigned `velox.exe` and `velox-host.exe` |
| Windows can identify and validate the publisher | SignPath Foundation Authenticode | Signed `velox.exe` and `velox-host.exe` |
| The expected GitHub workflow produced the distribution | GitHub artifact attestation | Final signed release ZIP |
| The SBOM belongs to that distribution | GitHub artifact attestation | Final SPDX SBOM |
| The signed output descends from the verified unsigned input | Repository-owned signing record | Unsigned digests, signing request, signed digests, final digest |

No row substitutes for another. A checksum detects a mismatch only when its
own distribution is trusted. An Authenticode signature does not identify the
source workflow. An artifact attestation does not add a Windows publisher.

## Provider Decision

SignPath Foundation is the constrained first choice for public alpha because it
offers an open-source signing path without giving the project custody of a
certificate private key. Adoption remains conditional on SignPath accepting
the project and approving a policy that limits subjects to the two expected PE
files from the protected release workflow.

Microsoft Artifact Signing is the migration candidate when the project needs a
project-owned legal publisher identity or predictable paid operation. A local
PFX, self-signed certificate, and repository-held private key are not fallback
paths.

The current SignPath Foundation GitHub connector requires the signing input to
be a GitHub workflow artifact and all preceding jobs in that trusted build to
run on GitHub-hosted runners. The signing workflow must preserve that chain; it
cannot splice in an unverified self-hosted build step.

## Credential Boundary

- No certificate private key, PFX, recovery copy, or password is stored in the
  repository, GitHub Actions secret storage, workflow artifact, cache, or log.
- SignPath owns private-key custody and performs the signing operation.
- The current SignPath GitHub integration uses a submission API token. It is
  scoped to the signing project and kept only in the protected `alpha-signing`
  GitHub environment.
- Pull requests, forks, ordinary branch builds, and benchmark workflows cannot
  access the signing environment.
- The environment requires maintainer approval. Repository workflow approval
  and provider signing approval are separate gates.
- Token exposure stops signing immediately and triggers token revocation,
  workflow audit, and preservation of affected request IDs.

## Artifact Flow

1. Check out an immutable release tag and record commit, workflow, runner, and
   dependency identities.
2. Build the CLI and host twice in independent clean workspaces.
3. Require byte-identical unsigned `velox.exe` and `velox-host.exe` digests.
4. Emit unsigned checksums, SBOM, and source-to-unsigned provenance. Preserve
   these as evidence; do not publish them as the final distribution.
5. Package only the two unsigned executables as the signing request input and
   record its digest.
6. Submit the exact input to the approved SignPath project, artifact
   configuration, and signing policy.
7. Download the provider output and require exactly the two expected file
   names. Reject added, removed, or renamed entries.
8. Verify Authenticode policy, certificate chain, expected publisher/profile,
   digest algorithm, and trusted timestamp for both executables.
9. Build the final release bundle from those exact signed executables. Generate
   host metadata and the release manifest from the signed bytes.
10. Build the final bundle twice from the same signed inputs and require
    byte-identical ZIPs. This proves deterministic assembly, not reproducible
    signing.
11. Emit final checksums, SPDX SBOM, and a signing record that binds unsigned
    input, provider request, signed output, and final bundle digests.
12. Create GitHub artifact attestations for the final signed ZIP and final SBOM.
13. Verify the complete candidate before publishing it as one immutable GitHub
    Release.
14. From an independent workflow, download the public asset and verify its
    checksum, attestations, executable signatures, manifest, consumer build,
    inspection, and startup.

## Signing Record

The future machine-readable signing record must contain at least:

- schema version and release version;
- source repository, commit, tag, workflow identity, and workflow run ID;
- unsigned executable names, sizes, and SHA-256 digests;
- signing provider, project, artifact configuration, policy, and request ID;
- signed executable names, sizes, and SHA-256 digests;
- certificate subject, issuer, serial or provider certificate identifier, and
  timestamp authority identity;
- final release ZIP name, size, and SHA-256 digest;
- final release manifest, checksum file, SBOM, and attestation identities.

The record must not contain an API token, certificate private material, raw
OIDC token, environment secret, or unredacted provider response.

## Workflow Permission Boundary

The workflow default is read-only. Jobs receive only the permission they need:

- build and signing-submission jobs do not receive release-write permission;
- the attestation job receives `id-token: write` and `attestations: write` only
  for final subjects;
- the publication job receives `contents: write` only after all verification
  and protected-environment gates pass.

No mutable action tag is accepted in the implementation. Every action version
must be live-checked against its official upstream source and pinned by commit
SHA at the time it is added or updated.

## Failure and Stop Conditions

Stop the candidate without publication when any of these occurs:

- unsigned independent-build digests differ;
- the signing request digest differs from the verified unsigned input;
- provider project, configuration, policy, or returned file set differs;
- either signature, chain, digest algorithm, publisher/profile, or timestamp
  verification fails;
- final manifest or SBOM does not describe the signed bytes;
- final artifact or SBOM attestation is missing or names another digest;
- a workflow attempts to rebuild between verification and promotion;
- the release tag or asset name already exists;
- provider credentials appear in output, artifacts, cache, or logs.

Failed candidates remain unpublished. Evidence needed for diagnosis is retained
without credentials; signing inputs and outputs follow the provider and GitHub
retention policy selected during implementation.

## Promotion and Rollback

Promotion moves one already verified final artifact between release channels.
It never rebuilds, re-signs, renames over, or replaces bytes. Alpha, beta, and
stable references must resolve to an immutable release version and digest.

A bad release is withdrawn from recommendation and replaced by a new version.
Existing assets and tags are preserved for investigation. Certificate
compromise, revocation, or provider-policy breach also triggers provider
suspension, API-token revocation, affected-version identification, and a public
security notice when consumers may be exposed.

## Implementation Gate

Do not add the signing workflow until all of these external values exist:

- accepted SignPath Foundation project;
- approved artifact configuration and signing policy;
- expected publisher/profile identity;
- confirmed GitHub-hosted trusted-build topology;
- protected GitHub environment and named approver;
- provider API-token scope and rotation owner;
- tested signature-verification command path;
- repository-owned signing-record schema and validator.

The first implementation must support a no-publication dry run and prove its
lineage checks before it receives release-write permission.
