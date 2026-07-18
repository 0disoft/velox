# ADR 0014: Adopt Actutum as the public identity

- Status: Accepted
- Date: 2026-07-18
- Owner: Project maintainer
- Supersedes: ADR 0013's open replacement-name gate

## Context

ADR 0013 blocks the first public executable under the Velox working name. The
name collides directly with an established Meta project and a released Go CLI
that already installs `velox` and `velox.exe`.

The project needs one identity that works as a product name, an ASCII shell
command, a Windows executable, a Go module path, a release-artifact prefix, and
a developer-search term. There is no public release or supported consumer of
the working-name contracts, so preserving those identifiers would create two
names before the first user exists.

## Decision

Adopt **Actutum** as the public product name. The Latin adverb *actutum* means
"immediately", "instantly", or "without delay". Use `actutum` as the canonical
ASCII spelling and shell command.

The public identity consists of:

- product: `Actutum`;
- CLI and executable: `actutum` and `actutum.exe`;
- generic host: `actutum-host.exe`;
- Go module and intended repository: `github.com/0disoft/actutum`;
- project manifest: `actutum.json`;
- runtime configuration: `actutum.runtime.json`;
- JavaScript bridge: `window.actutum`;
- environment prefix: `ACTUTUM_`;
- schema and evidence namespace: `actutum.*` and
  `https://schemas.actutum.invalid/`;
- release prefix: `actutum-windows-x64`.

The unpublished `0.5.10-alpha.1` Velox candidate is abandoned. The first
Actutum candidate is `0.6.0-alpha.1`. This minor-version change is deliberate:
the module, command, filenames, environment variables, bridge, and schemas all
change together before any compatibility promise exists.

Do not provide aliases for the working-name command, files, bridge, environment
variables, or schema IDs. Historical ADRs and preserved benchmark artifacts may
still say Velox when describing what actually existed at that time.

## Evidence boundary

The 2026-07-18 exact-name screen found:

- zero GitHub repositories and users named `actutum`;
- no exact npm, crates.io, PyPI, NuGet, or pkg.go.dev package;
- no active software product or CLI in general developer search;
- one unrelated Czech company outside the software field.

This is a bounded developer-namespace screen, not trademark clearance, legal
advice, a domain reservation, or a guarantee that a name remains available.

## Consequences

- Every maintained source, schema, fixture, workflow, release contract, and
  public document must move atomically to the new identity.
- The GitHub repository must be renamed before publication so the selected Go
  module path and public links resolve without relying on a stale path.
- Release evidence must be rebuilt because executable names, metadata, and
  archive bytes change.
- SignPath remains deferred. A future application must use the final public
  identity and an already released eligible artifact.
- Historical evidence is not relabeled. A historical Velox result and a new
  Actutum result are different evidence records even when they test the same
  implementation lineage.

## Alternatives

### Keep Velox and change only the executable

Rejected. It leaves search, repository, package, and support ambiguity while
creating a second command identity users must learn.

### Use Actutum only as a marketing name

Rejected. A split marketing and technical identity recreates the same package
discovery problem ADR 0013 was written to stop.

### Preserve working-name compatibility aliases

Rejected. No public release depends on them. Aliases would permanently widen
the command, configuration, environment, bridge, schema, and testing surface
without preserving a real user contract.

## Exit criteria

- Maintained source contains the old name only in explicit historical or
  collision-review contexts.
- The Go module, command directories, binaries, manifests, schemas, bridge,
  environment variables, workflows, and release evidence use Actutum.
- Focused rename hygiene, full tests, lint, workflow validation, release build,
  and clean-consumer smoke pass.
- `0disoft/actutum` is claimed before a tag or public release is created.
- The renamed candidate digest is recorded by the release evidence pipeline.

## Revisit

Revisit before commercial distribution, trademark registration, a package-
manager submission, or a move into a jurisdiction where the unrelated company
creates material confusion. A future rename requires a new ADR and cannot
rewrite historical evidence.
