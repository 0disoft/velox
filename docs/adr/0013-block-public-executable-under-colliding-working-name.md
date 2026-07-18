# ADR 0013: Block the public executable under the colliding working name

- Status: Superseded by ADR 0014
- Date: 2026-07-18
- Owner: Project maintainer

## Context

Risk R-011 requires a naming review before the first public package release.
The review found that `Velox` is not merely common Latin-derived branding:

- Meta operates the established Velox data execution engine and ecosystem.
- A released Go speed-test CLI already distributes the exact `velox` command
  and `velox.exe` on Windows.
- Active TypeScript, Go, and database projects use the same root name.

This project is itself a Go CLI for Windows. The existing executable collision
creates concrete PATH, package-discovery, support-search, and reputation
confusion even if trademark classes or product purposes differ.

## Decision

Keep `Velox` as the working repository and historical evidence name, but do not
publish the first downloadable executable under that identity.

The selected internal candidate version remains `0.5.10-alpha.1`. Creating its
public tag and release is blocked until the maintainer selects a replacement
name and the rename synchronizes executable, command, package, schema,
environment, profile, workflow, evidence, and documentation contracts.

The collision scan is product evidence, not legal advice. A future decision to
retain the word requires a new ADR and at minimum a non-colliding executable
command.

## Consequences

- The unsigned-preview pipeline remains implemented and testable but dormant.
- No SignPath or signing work resumes because naming precedes publisher
  identity.
- Historical benchmark artifacts remain valid and are not rewritten.
- Rename work must distinguish public identifiers from internal protocol
  compatibility instead of using an indiscriminate text replacement.
- M4 remains active until rename, publication, public download verification,
  and independent external use are complete.

## Alternatives

### Publish first and rename later

Rejected. The first external users would install a known-colliding command and
all release links, hashes, screenshots, and support reports would immediately
become migration debt.

### Qualify only the README title

Rejected. `Velox Desktop Packager` does not solve the exact `velox.exe` and
shell-command collision.

### Treat different product domains as sufficient separation

Rejected for engineering identity. Even if legally permissible, command and
search collisions remain observable user problems.

## Exit Criteria

- A replacement is explicitly selected by the maintainer.
- `docs/product/05-naming-review.md` is refreshed for that candidate.
- A rename plan classifies compatibility-sensitive and historical identifiers.
- Tests, workflows, release bundles, docs, examples, and benchmark adapters
  agree on the public identity.
- The release candidate is rebuilt and its new digest is recorded before tag
  creation.

ADR 0014 selects Actutum and owns the replacement and compatibility decision.
