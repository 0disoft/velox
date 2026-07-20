# ADR 0016: Separate technical distribution from independent adoption

- Status: Accepted
- Date: 2026-07-20
- Owner: Project maintainer
- Supersedes: the M4 completion criterion in ADR 0011 and ADR 0012

## Context

ADR 0011 and ADR 0012 made a qualifying independent-user attempt the final M4
gate. That mixed two different questions:

1. Can a clean Windows runner acquire the published bytes and complete the
   documented compile-free path without private source or maintainer tooling?
2. Does a person outside the project find the product valuable enough to try?

The first question is an engineering and distribution contract the project can
verify. The second is adoption evidence controlled by people outside the
project. A maintainer cannot manufacture independence by moving the same test
to another branch, account, or repository. Keeping M4 blocked on an actor the
project does not control also creates pressure to mislabel internal automation
as external validation.

## Decision

M4 is the technical alpha-distribution milestone. It completes when a separate
public consumer repository controlled by the maintainer proves all of the
following from a GitHub-hosted clean runner:

1. It downloads one pinned public Velox release and verifies the independently
   recorded release ZIP SHA-256.
2. It does not check out Velox source or invoke a compiler, Node.js, frontend
   package manager, or GitHub Actions cache.
3. It exercises `version`, `init`, `validate`, `doctor`, two deterministic
   `build` operations, `inspect`, and `run` through the public release CLI.
4. It validates checksums, SBOM, provenance metadata, release identity,
   deterministic output, inspection, and startup.
5. Its evidence schema fixes `maintainerControlled: true` and
   `externalUserAttempt: false`.

The public [`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository satisfies that contract at commit
`ed003602d65cbaef12bf95ee78b2cf16466bdfcd`. Hosted
[run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250)
completed on `windows-2025`. Artifact `8458382152` has digest
`sha256:0b2438041e312a49c934d0dd89676c0bf85d4404b13caef4956a7ee51295e0c4`.
The schema-valid result records all checks true, no consumer toolchain command,
zero Actions cache upload bytes, release source commit
`9f10c545b6bde23d2c3dad5bbb12bffdac513712`, and release ZIP SHA-256
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`.

After recording that one-shot evidence, the consumer repository was archived.
It remains public and read-only so the workflow source, commit, and run link are
preserved. Future release verification belongs to the main Velox repository;
the archived lock and result must not be advanced to another release.

M4 is therefore complete.

Independent-user attempts remain a separate M5 adoption input. The current
count is zero. M5 may begin with that absence recorded, but the project cannot
claim independent validation, adoption, documentation usability, or user
demand until qualifying evidence exists. A decision to enter beta or a broader
support channel must either obtain such evidence or explicitly accept the
zero-adoption risk in a later ADR.

Signing remains adoption-triggered future-channel work. Maintainer-controlled
consumer success does not activate SignPath or Authenticode work by itself.

## Alternatives

### Keep M4 blocked until an independent person appears

Rejected. It makes a technical milestone depend on an uncontrolled social
event and prevents the M5 stop, continue, or reposition decision from even
starting when adoption is absent.

### Label the separate repository as an external user

Rejected. Repository separation improves the source and toolchain boundary but
does not create an independent owner. The evidence must continue to say
`externalUserAttempt: false`.

### Remove external-user evidence from the roadmap

Rejected. Lack of independent use is important product evidence. It belongs in
the product decision, not in the technical distribution proof.

## Consequences

### Positive

- M4 now has an objective, reproducible completion boundary.
- The clean-room consumer path cannot silently consume unpublished source.
- M5 can evaluate stopping or repositioning when no external user exists.
- Public wording cannot confuse same-owner automation with adoption.

### Negative

- M4 completion does not prove product demand or documentation usability.
- The hosted artifact expires under GitHub retention policy and may need to be
  rerun for a later audit; the immutable run, commit, and artifact digest remain
  the current receipt.
- A positive beta decision still needs evidence beyond maintainer automation or
  an explicit acceptance of the resulting market risk.

## Validation

The consumer repository owns its release lock, workflow, evidence schema, and
fail-closed contract checks. The hosted evidence must match the exact consumer
commit, workflow run ID and attempt, release source commit, and release digest.

Velox hygiene tests require the M4/M5 status, the consumer run identity, and the
non-adoption wording to stay synchronized across the roadmap, product spec,
validation guide, release operations, and external-attempt contract.

## Rollback or Fallback

If a rerun cannot reproduce the result, reopen M4 and record the failing phase.
Do not replace the original run or relabel a failed result. Publish a corrected
consumer commit and cite a new immutable run.

If the public release disappears or its bytes change, treat that as a release
integrity incident rather than satisfying the gate with a local copy.

## Revisit Triggers

- A qualifying independent-user attempt is accepted.
- M5 chooses beta, stable, or broader support.
- The public release or consumer repository changes ownership.
- Distribution moves away from GitHub Releases.
- The consumer workflow starts requiring another toolchain or cache.

## Synchronized Surfaces

- `README.md`
- `VALIDATION.md`
- `docs/README.md`
- `docs/adr/README.md`
- `docs/ops/00-operational-contract.md`
- `docs/ops/external-user-attempt.md`
- `docs/ops/release.md`
- `docs/product/01-roadmap.md`
- `docs/product/02-spec.md`
- `docs/product/03-risk-register.md`
- `docs/product/05-naming-review.md`
- `docs/engineering/08-m4-security-review.md`
- hygiene tests
