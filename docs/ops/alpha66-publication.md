# Alpha.66 Publication: 2026-10-05

This is the publication record for unsigned prerelease `v0.5.10-alpha.66`. It
records the public release identity, the reused tag-CI assets, the narrow
unauthenticated public download checks, and the scopes deliberately not run.

## Release Identity

- Tag: `v0.5.10-alpha.66`.
- Release URL: https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.66.
- Release ID: `403770595`; `prerelease: true`, `draft: false`, `--latest=false`.
- Published: `2026-10-05T13:56:59Z` (client day 2026-10-05).
- Source: `842fead0c889e9f161c2567a91c8d0fd4c2ca260`; the annotated tag was not
  moved.
- Method: `gh release create --prerelease --verify-tag --latest=false`, using
  the existing tag-CI assets only.

## Tag CI Input

[Tag CI 37317775399](https://github.com/0disoft/velox/actions/runs/37317775399)
attempt 1 (`event: push`, conclusion `success`, 13:33:52Z-13:35:16Z) produced
the four assets with Go `1.26.0 windows/amd64` on `windows-2025`. The
producer job `111788772952` ran release-contract tests and two release builds;
the checkout-free consumer job `111789306289` invoked only `velox.exe` and
reported two matching app ZIPs. The `Publish unsigned preview` job
`111789383802` was skipped in that run. Publication reused these exact
verified assets with no second producer and no rebuild; the CI run was not
rerun. See [the alpha.66 tag evidence](alpha66-tag-evidence.md).

## Published Assets

| Asset | Asset ID | Bytes | SHA-256 |
| --- | --- | --- | --- |
| `velox-windows-x64.zip` | 612727255 | 5,903,850 | `3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc` |
| `checksums.sha256` | 612727257 | 279 | `f20db6a79ad78170d83360b4b067d1024ca20220eaaae40c76c15c381728dd72` |
| `velox-windows-x64.spdx.json` | 612727254 | 9,618 | `196d958fbd52c8c68a10d75a7da7d9c6896fbb15ac5dbdb8b2fd46a2f3c26938` |
| `velox-windows-x64.intoto.jsonl` | 612727256 | 744 | `951fabec910eadb00d92a94fed0fa358ea328a39868fe846aee558410d3fe1c2` |

Release executables inside the ZIP, from `verified-result.json`, matching the
manifest entries:

| File | Bytes | SHA-256 |
| --- | --- | --- |
| `velox.exe` | 4,550,656 | `e04309c072c3d4ce96c2dae485e338b3dff225b91b352d6474edb4ccbbd8888b` |
| `velox-host.exe` | 4,416,000 | `7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb` |
| `velox-setup.exe` | 3,574,784 | `e0b40362dd0c82a8b31ea4922025683f952a372167d3086117fab1b44de86786` |

## Public Download Verification

All four `browser_download_url` files were fetched once without Authorization
headers or credentials (`acquisition` = public HTTPS `browser_download_url`;
`externalUserAttempt: false`). For every asset the downloaded bytes, sizes and
SHA-256 matched the CI and staged copies and the GitHub release metadata
digests. The three `checksums.sha256` entries, the 16 manifest artifacts, the
17 SPDX file digests, ZIP/CRC, version/target and the provenance
subject/source/invocation `37317775399/1` all matched. This is
`same-repository-public-download` evidence with `externalUserAttempt: false`,
not independent adoption.

The narrow public CLI version check expanded the downloaded ZIP to
`dist/candidates/alpha66-ci-37317775399/public-extracted` and ran only
`velox.exe version --json`. It exited 0 and reported version
`0.5.10-alpha.66`, matching `internal/buildinfo`. This check does not exercise
public app packaging or any native UI. Receipts: `publication-result.json`,
`published-release.json`, `verified-result.json`, `public-cli-version.json`
and `run-result.json` under
`dist/candidates/alpha66-ci-37317775399/`.

## Scope Not Run

This publication verified artifact identity, the public CLI version, and the
in-repo release-contract and checkout-free consumer jobs. It did not run, and
does not claim, any of the following against the published bytes:

- Native UI, window startup or OS interaction.
- Installer execution.
- `run --watch` or draft recovery.
- Multi-template packaging.
- A hosted public verifier dispatch or an external-user attempt.
- Performance, stress or permission-matrix measurement.

The prior implementation and local native evidence for these areas is retained
in its own records but is not upgraded to this publication. No pass is claimed
for a scope that did not run.

## Contract Boundary

This publication record includes no API, database/schema, dependency or
runner/job-scheduling change. The executables are unsigned; the checksum file,
SPDX SBOM and provenance statement describe the release files but provide no
Windows publisher identity, and the provenance statement is not an
authenticated attestation. The release is not a beta promotion or a
human-adoption claim.
