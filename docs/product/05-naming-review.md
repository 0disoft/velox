# Public Name Selection Review

- Status: Velox retained by maintainer decision
- Date: 2026-07-19
- Owner: Project maintainer
- Selected public name: Velox
- Selected CLI spelling: `velox`

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

## Maintainer decision

The selected public name remains **Velox**. The public command is `velox`, the
Windows CLI is `velox.exe`, and the generic host is `velox-host.exe`. The Go
module and repository remain `github.com/0disoft/velox`. The first public
unsigned developer preview is `0.5.10-alpha.1`.

This decision accepts the developer-discovery, PATH, package, and support-search
risks described above. It does not deny those collisions and is not a legal or
trademark opinion. ADR 0015 is the binding decision.

## Rejected replacement

Actutum was screened as a possible replacement and briefly applied across the
source and benchmark repositories, but the project maintainer never approved
changing the product away from Velox. ADR 0014 and Git history retain that
mistaken transition as historical evidence. Actutum is not an alias, package
name, command, executable, schema namespace, or migration target.

## Compatibility decision

No public release was created during the attempted rename. Maintained source,
workflows, schemas, examples, benchmark adapters, and release evidence therefore
return directly to their Velox identifiers without compatibility aliases.
Published historical benchmark artifacts remain byte-for-byte unchanged. The
later Velox release was published only after ADR 0015 restored and fixed the
public identity.

## Publication gate

The name decision no longer blocks the developer preview, and public unsigned
preview `v0.5.10-alpha.1` has been published with reproducible producer,
checkout-free consumer, and same-repository public-download evidence. A
qualifying independent external-user attempt remains the M4 completion gate; it
is not a publication prerequisite. Formal trademark and distribution-channel
review remain external gates if commercial or package-manager distribution
begins.
