# ADR 0006: Bind the CLI and host with release metadata

- Status: Accepted
- Date: 2026-07-11
- Owner: Project maintainer

## Context

The compile-free build copies a generic host without changing its bytes. A file
named `actutum-host.exe` is not enough compatibility or integrity evidence: an
older, newer, wrong-target, or modified host could otherwise be packaged by a
different CLI release.

## Decision

Each Windows x64 release bundle places `actutum-host.json` beside `actutum.exe` and
`actutum-host.exe`. The sidecar uses `actutum.host/v1` and binds:

- Actutum release version;
- target;
- host contract version;
- runtime configuration contract version;
- exact host filename, byte count, and SHA-256.

`validate` and `build` fail with the host-compatibility exit class unless every
field agrees with the running CLI and selected host. There is no public option
to substitute arbitrary host metadata or bypass verification.

The release-bundle assembler is maintainer-only. It compiles the pure-Go CLI
and host, generates metadata and a release manifest, copies schemas and notices,
and writes a deterministic unsigned ZIP. It performs no network, signing,
publishing, or deployment action.

## Consequences

- Consumer builds detect accidental host and CLI skew before output mutation.
- The host executable remains byte-identical and externally signable.
- Release metadata is inspectable and versioned independently from Go types.
- The sidecar protects consistency only after the complete bundle is trusted.
  It is not a signature and cannot defeat an attacker who replaces both the
  host and metadata.
- Public alpha still requires checksums, SBOM, provenance, signing, and
  post-download verification.

## Rollback

Rollback selects a previous complete immutable release bundle. Mixing a CLI,
host, or sidecar from different releases is deliberately rejected.
