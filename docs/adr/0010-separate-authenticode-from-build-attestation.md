# ADR 0010: Separate Authenticode from build attestation

- Status: Accepted for future signed channels; M4 gate superseded by ADR 0011
- Date: 2026-07-18
- Owner: Project maintainer

## Context

M4 needs two different trust claims. A consumer must be able to verify that a
release came from the expected repository workflow, and Windows must be able to
verify the publisher of each distributed executable. Neither claim implies the
other.

GitHub artifact attestations use the workflow identity and artifact digest to
authenticate build provenance. They do not add an Authenticode publisher
signature to a PE file. Authenticode signs the executable bytes through a
certificate provider, but it does not by itself prove which repository commit
and workflow produced the unsigned input.

The current alpha-evidence workflow proves that two unsigned release builds are
byte-identical. It emits unsigned provenance, but no authenticated attestation
or executable signature. Signing also changes PE bytes, so a signed release
cannot inherit the unsigned archive digest or the claim that independent
signing runs are byte-identical.

## Decision

Use two independent controls when Velox opens a signed distribution channel:

1. GitHub artifact attestations authenticate the final distribution artifact
   and its SBOM through GitHub OIDC and Sigstore.
2. SignPath Foundation is the first Authenticode provider candidate,
   contingent on project acceptance and an approved signing
   policy. SignPath holds the private signing key; Velox stores no certificate
   private key or PFX.

Microsoft Artifact Signing is the migration candidate when Velox needs its own
legal publisher identity, paid service terms, or a less approval-dependent
signing path. Provider-specific workflow code must remain an adapter around the
repository-owned signing contract in `docs/ops/signing.md`.

The release lineage has two stages:

- reproducible unsigned executables, with independent-build equality evidence;
- provider-signed executables and a deterministic final bundle assembled from
  those exact signed inputs.

The final release manifest records the signed executable digests. The
authenticated artifact attestation names the final signed ZIP, not the unsigned
predecessor. A separate signing record must bind the unsigned executable
digests, provider request and policy identity, signed executable digests, and
final bundle digest.

Only `velox.exe` and `velox-host.exe` are Authenticode subjects. The ZIP is
protected by checksum and artifact attestation. Application executables created
by Velox remain the application publisher's signing responsibility.

## Alternatives

### GitHub artifact attestations only

Rejected. They authenticate workflow provenance but do not create a Windows
publisher signature.

### Authenticode only

Rejected. A valid publisher signature does not bind the artifact to the
expected source commit and GitHub workflow.

### Repository or GitHub secret containing a PFX

Rejected. It makes private-key custody, rotation, extraction resistance, and
incident response a solo-maintainer responsibility.

### Microsoft Artifact Signing for the first alpha

Deferred. It is the preferred migration path for a project-owned publisher
identity, but it adds paid account, identity-validation, role, and certificate-
profile operations before the first open-source alpha.

### Separate private release-operations repository

Deferred. It would add a cross-repository artifact handoff before the signing
contract exists. The public repository owns non-secret policy and workflow;
provider credentials and approvals belong to a protected GitHub environment.

## Consequences

### Positive

- Provenance and Windows publisher identity can be verified independently.
- No signing private key enters the repository or GitHub secret storage.
- The unsigned reproducibility claim remains honest after timestamped signing.
- A provider migration does not redefine release-manifest or lineage semantics.

### Negative

- A future signed channel depends on SignPath Foundation acceptance and service
  availability.
- A signing request can require manual approval and add release latency.
- The current SignPath Foundation GitHub trust path requires the preceding
  build jobs to run on GitHub-hosted runners.
- The final signed ZIP is not reproducible across independent signing requests.
- The release pipeline must preserve and verify both unsigned and signed
  evidence instead of treating signing as an opaque final command.

## Validation

This ADR is a design decision, not completed signing evidence. ADR 0011 removes
these checks from the unsigned M4 developer-preview gate. A future signed
channel remains incomplete until an implemented workflow proves all of the
following:

- two unsigned builds produce byte-identical executables;
- the signing provider receives the recorded unsigned digests;
- both returned executables pass Authenticode chain, profile, and timestamp
  verification;
- the final release manifest matches the signed executable bytes;
- the final ZIP and SBOM receive GitHub artifact attestations;
- an independent workflow downloads and verifies the public release.

Every third-party GitHub Action used by that workflow must be checked against
its current upstream release and pinned by immutable commit SHA when the
workflow is implemented.

## Rollback or Fallback

If SignPath onboarding fails, the signed channel remains blocked; the project
does not fall back to a repository-stored key. The maintainer may adopt
Microsoft Artifact Signing through a superseding ADR after identity, cost,
roles, and dry-run evidence are available.

If signing or attestation fails after an unsigned build, discard that signed
candidate and preserve the unsigned evidence. ADR 0011 separately permits an
explicitly labeled unsigned developer preview. Never relabel unsigned bytes as
signed or replace an existing release asset.

## Revisit Triggers

- SignPath Foundation rejects or discontinues the project.
- The certificate publisher identity is unsuitable for intended users.
- Manual approval or provider availability makes releases impractical.
- Microsoft Artifact Signing identity and cost prerequisites become acceptable.
- Velox starts signing application-specific executables.
- GitHub changes artifact-attestation availability or identity semantics.

## Synchronized Surfaces

- `docs/adr/README.md`
- `docs/ops/signing.md`
- `docs/ops/release.md`
- `docs/ops/rollback.md`
- `docs/product/01-roadmap.md`
- `docs/product/03-risk-register.md`
- `docs/engineering/04-security-baseline.md`

## References

- [GitHub artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations)
- [SignPath Foundation](https://signpath.org/)
- [SignPath GitHub trusted build integration](https://docs.signpath.io/trusted-build-systems/github)
- [Microsoft Artifact Signing integrations](https://learn.microsoft.com/en-us/azure/artifact-signing/how-to-signing-integrations)
