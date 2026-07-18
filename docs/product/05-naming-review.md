# Public Name Selection Review

- Status: Replacement selected and source rename verified locally; repository rename pending
- Date: 2026-07-18
- Owner: Project maintainer
- Selected public name: Actutum
- Selected CLI spelling: `actutum`
- Intended pronunciation: `ak-TOO-tum`

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

## Original-name decision

`Velox` is not approved as the public executable or final product identity.
It remains only in historical evidence and the ADRs that explain why the name
was replaced.

## Selected replacement

The selected replacement is **Actutum**. The Latin adverb *actutum* means
"immediately", "instantly", or "without delay". It keeps the original speed
idea while pointing at the product's actual promise: minimal setup and build
work before a static desktop application can run.

The public command is `actutum`, the Windows CLI is `actutum.exe`, and the
generic host is `actutum-host.exe`. The next candidate is
`0.6.0-alpha.1`; the unpublished `0.5.10-alpha.1` working-name candidate will
not be tagged or released.

This is a product and developer-namespace decision, not a legal opinion.

## Replacement evidence

The 2026-07-18 screen checked the selected ASCII spelling against these
surfaces:

1. GitHub repository-name and user-name search returned no exact `actutum`
   match.
2. npm, crates.io, PyPI, NuGet, and pkg.go.dev returned no exact package.
3. General developer search found no active software product or CLI using the
   exact name.
4. General company search found an unrelated Czech company whose listed fields
   are wholesale, retail, property, advertising, and administration rather
   than software.

The search does not reserve a namespace. The GitHub repository rename must
claim `0disoft/actutum` before the first public release. Homebrew, Winget,
Scoop, Chocolatey, domain, and formal trademark availability remain publication
or distribution-channel checks because no package is being submitted to those
channels now.

The selected name must have a written meaning, intended pronunciation, and
stable ASCII CLI spelling. Search uniqueness matters more than preserving the
Latin speed metaphor.

## Compatibility decision

No public release exists, so the working-name contracts are not a compatibility
surface. The implementation rename must update in one atomic review:

- repository title and public release artifact names;
- `actutum.exe`, `actutum-host.exe`, and maintainer helper names;
- Go module path to `github.com/0disoft/actutum` together with the GitHub
  repository rename;
- CLI command text, JSON envelopes, environment variable prefix, schemas and
  schema IDs;
- application profile directory, virtual-origin labels, examples, docs, issue
  templates, workflows, benchmark adapters, and release evidence;
- compatibility policy for the working-name alpha candidate.

Protocol identifiers, schema IDs, environment variables, the JavaScript bridge,
default manifest names, and runtime filenames all move to `actutum` because no
published consumer depends on the working-name forms. Historical benchmark
artifacts and ADR explanations are not rewritten as if they had been produced
under the new name.

## Local implementation evidence

The renamed `0.6.0-alpha.1` source produced a deterministic local
`actutum-windows-x64.zip` candidate on 2026-07-18:

- archive bytes: `3224390`;
- archive SHA-256:
  `6df8d7ad2a81c9432dc53b2c16cd0c94a227ab718d8d3e8648c269c42fb315b7`;
- SPDX SHA-256:
  `134501655a699f217d8ac69717f48f6bc75320a644734b46b4292fa202336fb4`;
- unsigned provenance was emitted for the same archive and source commit. Its
  file hash is intentionally invocation-specific because the statement records
  the evidence run's `invocationId`; it is not a reproducibility anchor.

The compiler-free consumer smoke initialized `dev.actutum.*`, built twice,
inspected the resulting ZIP, and the renamed host reached its ready callback.
These are local verification results. They are not a GitHub Release, hosted
same-commit evidence, an external-user attempt, a signature, or an
authenticated attestation.

## Publication gate

Do not publish until the source rename is complete, the GitHub repository owns
the selected slug, release evidence has been rebuilt from the renamed source,
and current exact-name searches are repeated immediately before publication.
Formal trademark review remains external if commercial use begins.
