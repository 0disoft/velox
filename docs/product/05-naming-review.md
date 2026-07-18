# Public Name Collision Review

- Status: Complete collision scan; replacement name required before public executable release
- Date: 2026-07-18
- Owner: Project maintainer
- Current working name: Velox

## Scope

This is a developer-discoverability and package-identity scan. It is not a
trademark opinion, legal clearance, domain purchase, or worldwide registry
search.

The relevant question is narrower: would a developer reasonably identify the
project, install its CLI, and invoke its executable without colliding with an
existing software product?

## Direct Collisions

### Meta Velox

[Meta Velox](https://velox-lib.io/) is an established open-source execution
engine for data systems, with its own project site, community, conference, and
large ecosystem. It dominates developer search results for the bare word
`Velox`.

The products operate in different domains, but a qualifier would be required in
nearly every search, issue, package description, and comparison. That is a
discoverability cost, not merely a legal question.

### Existing Go CLI and `velox.exe`

[`github.com/koller-nexus/velox`](https://pkg.go.dev/github.com/koller-nexus/velox)
is a released Go internet-speed-test CLI. Its documented Windows installation
ships `velox.exe`, and its Go installation command places a `velox` command on
`PATH`. Version `v1.0.1` was published in July 2026.

This is a direct command and executable collision with the current product,
which is also a Go CLI intended to ship `velox.exe` on Windows. Different
purposes do not prevent PATH, package-manager, support-search, and security-
reputation confusion.

### Other active software namespaces

[`@veloxts/velox`](https://www.npmjs.com/package/@veloxts/velox) is an active
TypeScript framework and CLI family. Other Go modules and database products
also use the name. None is individually decisive, but together they show that
the bare namespace is crowded.

## Decision

`Velox` remains a repository working name and historical benchmark label. It is
not approved as the public executable or final product identity.

Do not create the first public downloadable executable release until a
replacement name has been selected and checked. The internal candidate version
`0.5.10-alpha.1` remains valid; its tag and artifact names are not published
until the rename contract is synchronized.

## Replacement Gate

Before publication, verify the selected name against all of these surfaces:

1. GitHub repository and organization search.
2. Exact Windows executable and common shell command names.
3. Go module and `go install` command search.
4. npm, crates.io, PyPI, Homebrew, Winget, Scoop, and Chocolatey search where
   future distribution is plausible.
5. General developer search and adjacent desktop-framework products.
6. Basic trademark and domain checks performed by the maintainer; legal advice
   remains external when commercial use is planned.

The selected name must have a written meaning, intended pronunciation, and
stable ASCII CLI spelling. Search uniqueness matters more than preserving the
Latin speed metaphor.

## Rename Surfaces

A replacement affects more than README prose. The rename change must update in
one atomic review:

- repository title and public release artifact names;
- `velox.exe`, `velox-host.exe`, and maintainer helper names as decided;
- Go module path only if the GitHub repository is renamed;
- CLI command text, JSON envelopes, environment variable prefix, schemas and
  schema IDs;
- application profile directory, virtual-origin labels, examples, docs, issue
  templates, workflows, benchmark adapters, and release evidence;
- compatibility policy for the working-name alpha candidate.

Do not perform a blind global replacement. Protocol identifiers and historical
evidence need an explicit compatibility decision.

## Revisit

Close this gate only with a named replacement and current collision evidence.
If the maintainer deliberately keeps `Velox`, record the legal and operational
acceptance in a new ADR and choose a non-colliding executable command anyway.
