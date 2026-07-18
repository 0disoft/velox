# ADR 0011: Publish an unsigned developer preview before signing

- Status: Accepted
- Date: 2026-07-18
- Owner: Project maintainer

## Context

M4 previously treated Authenticode signing and SignPath Foundation acceptance
as prerequisites for the first public alpha. That ordering confuses a Windows
distribution trust improvement with the ability to build and publish the
software. Windows can run an unsigned executable, although SmartScreen can
warn and managed-device policy can block unknown code.

Velox is still a developer-facing alpha. Its unanswered M4 question is whether
an external user can download the portable bundle and complete the documented
compile-free path. Provider onboarding does not answer that question and can
delay the first useful distribution evidence.

SignPath Foundation also requires an eligible project to be released in the
form it asks SignPath to sign. The repository may prepare a future provider
configuration, but provider application and signing workflow work should not
precede the first public preview.

## Decision

Publish the first public M4 artifact as an explicitly unsigned developer
preview. Authenticode signing is not an M4 publication gate.

The preview must:

1. use an immutable `vX.Y.Z-alpha.N` tag;
2. be built twice and pass byte-for-byte unsigned reproducibility;
3. pass the checkout-free consumer job before publication;
4. include the Windows x64 ZIP, SHA-256 checksums, SPDX SBOM, and unsigned
   provenance statement;
5. be marked as a GitHub prerelease;
6. state prominently that the executables are unsigned, SmartScreen can warn,
   managed devices can block them, and checksums do not create publisher
   identity;
7. refuse to replace an existing release or publish from a branch;
8. require an explicit manual publication confirmation.

The existing SignPath configuration, deterministic signing input, signing
record, and Authenticode verifier remain dormant preparation for a later
signed channel. They do not authorize provider submission or publication.

## Scope

This decision covers Velox's own developer-preview bundle. Applications built
with Velox remain unsigned unless their publisher signs them separately.

No installer, updater, Microsoft Store package, certificate purchase, provider
account, or application-signing API is added.

## Alternatives

### Require SignPath before any public artifact

Rejected for M4. It delays the external-user experiment, makes provider
acceptance part of product validation, and conflicts with SignPath's released-
project eligibility ordering.

### Publish binaries without integrity evidence

Rejected. Unsigned distribution still needs deterministic build evidence,
checksums, an SBOM, provenance metadata, and immutable release assets.

### Self-sign or store a PFX in repository automation

Rejected. Self-signing does not establish a broadly trusted publisher, and
repository-held private-key custody is unnecessary risk for an alpha.

### Make the preview look like a general-user stable release

Rejected. The expected warning and managed-device limitations would make that
positioning misleading.

## Consequences

### Positive

- M4 tests actual acquisition and use before spending effort on provider
  onboarding.
- Release evidence remains independent of signing-provider availability.
- The project can collect real SmartScreen and enterprise-policy friction.
- Future signing work retains the strict lineage and verifier already built.

### Negative

- Windows can show an unknown-publisher warning.
- Smart App Control or enterprise policy can block the executable.
- Checksums distributed beside the artifact do not protect against compromise
  of the entire GitHub release channel.
- The preview is unsuitable for users who require authenticated publisher
  identity.

## Validation

M4 publication is ready only when the configured release and policy checks
prove the workflow contract, reproducible bundle, evidence documents, and
checkout-free consumer path. M4 completes only after an independent user or
account downloads the public release and records the result.

Signing is reconsidered when at least one trigger is observed:

- repeated install abandonment or support requests caused by Windows warnings;
- enterprise or managed-device evaluation;
- automatic-update or installer work;
- expansion from developer preview to a general-user channel;
- a publisher identity requirement from a real adopter.

## Rollback or Fallback

A bad preview is marked unsuitable and replaced by a new immutable version.
The tag and assets are not overwritten. Documentation points users to the last
verified digest.

If unsigned execution blocks the intended audience before external product
value can be tested, resume ADR 0010's provider path. Do not weaken the warning
or fall back to a repository-held key.

## Synchronized Surfaces

- `README.md`
- `VALIDATION.md`
- `ARCHITECTURE.md`
- `docs/adr/README.md`
- `docs/adr/0010-separate-authenticode-from-build-attestation.md`
- `docs/ops/release.md`
- `docs/ops/rollback.md`
- `docs/ops/signing.md`
- `docs/ops/signpath-onboarding.md`
- `docs/product/01-roadmap.md`
- `docs/product/03-risk-register.md`
- `.github/workflows/alpha-evidence.yml`
- `tests/hygiene/consumer_workflow_test.go`

## References

- [Microsoft SmartScreen reputation](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/smartscreen-reputation)
- [Microsoft Smart App Control](https://learn.microsoft.com/en-us/windows/apps/develop/smart-app-control/overview)
- [Microsoft code-signing options](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/code-signing-options)
- [SignPath Foundation terms](https://signpath.org/terms.html)
