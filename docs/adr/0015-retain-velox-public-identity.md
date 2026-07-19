# ADR 0015: Retain Velox as the public identity

- Status: Accepted
- Date: 2026-07-19
- Owner: Project maintainer
- Supersedes: ADR 0013 and ADR 0014

## Context

ADR 0013 documented real developer-discovery and executable-name collisions
around `Velox`. ADR 0014 then selected Actutum and an implementation agent
applied that replacement across the source and benchmark repositories.

The project maintainer did not approve changing the product away from Velox.
The replacement recommendation was incorrectly promoted into a product
decision. No Velox or Actutum public release, supported package, or external
consumer contract was created during that mistaken transition.

## Decision

Retain **Velox** as the product and technical identity. Use these canonical
public identifiers:

- product: `Velox`;
- CLI and executable: `velox` and `velox.exe`;
- generic host: `velox-host.exe`;
- Go module and repository: `github.com/0disoft/velox`;
- project manifest: `velox.json`;
- runtime configuration: `velox.runtime.json`;
- JavaScript bridge: `window.velox`;
- environment prefix: `VELOX_`;
- schema and evidence namespace: `velox.*` and
  `https://schemas.velox.invalid/`;
- release prefix: `velox-windows-x64`.

The first unpublished candidate remains `0.5.10-alpha.1`. Actutum identifiers
are not compatibility aliases and must not remain in maintained source,
workflows, schemas, fixtures, or active product documentation. ADR 0014 and Git
history remain as the record of the rejected transition.

The known Meta project, existing Go CLI, and crowded search namespace remain
real risks. The maintainer accepts those risks for this project identity. They
must be disclosed in the naming review and revisited before package-manager or
commercial distribution, but they do not by themselves block the first
developer preview.

## Consequences

### Positive

- The repository, product, executable, module, benchmark, and release lineage
  retain the identity selected by the maintainer.
- No GitHub repository rename or split compatibility layer is required.
- Existing historical benchmark evidence remains correctly labeled Velox.

### Negative

- Search results remain dominated by unrelated Velox projects.
- The `velox` command and `velox.exe` collide with an existing Go CLI.
- Future package-manager publication may require a qualified package name or a
  separate distribution decision.

## Rejected alternatives

### Continue with Actutum

Rejected. It was never approved by the project maintainer.

### Keep Velox only as a repository nickname

Rejected. A different command or module identity would preserve the same split
that the mistaken rename created.

### Hide the collision review

Rejected. Retaining the name does not make the discovery and command risks
disappear.

## Validation

- Full source, lint, workflow, release-bundle, and clean-consumer checks pass
  with Velox identifiers.
- Maintained source contains Actutum only in ADR 0014 or explicit historical
  explanations of the rejected transition.
- The unsigned preview workflow targets `0disoft/velox` and is no longer
  disabled by ADR 0013's replacement-name gate.
- Benchmark adapters and active schemas use Velox while immutable historical
  evidence remains untouched.

## Revisit

Revisit before a package-manager submission, commercial distribution,
trademark registration, or verified user confusion that prevents adoption.
A future rename requires direct maintainer approval and a new ADR.
