# Alpha.66 Tag CI Evidence: 2026-10-05

## Identity

- Source: `842fead0c889e9f161c2567a91c8d0fd4c2ca260` on `main`.
- Tag: annotated `v0.5.10-alpha.66`; its peeled commit equals the same
  source SHA.
- Run: [37317775399](https://github.com/0disoft/velox/actions/runs/37317775399),
  attempt 1, `event: push`, conclusion `success` (13:33:52Z-13:35:16Z).
- Toolchain: Go `1.26.0 windows/amd64` (setup-go spec `1.26.0`,
  `go version go1.26.0 windows/amd64`, `GOTOOLCHAIN=local`). This differs
  from the prior local Go `1.27.1` candidate builds.
- Release ZIP: `velox-windows-x64.zip`, 5,903,850 bytes, SHA-256
  `3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc`.
- Not published. The tag run dispatched no publication; see Public Status.

The single tag push triggered one `alpha-evidence` workflow run on attempt 1;
no rerun or additional workflow dispatch was requested in this step.

## Jobs

| Job | ID | Result | Duration | Notes |
| --- | --- | --- | --- | --- |
| Build reproducible unsigned preview evidence | 111788772952 | success | ~63 s | release-contract tests, two release builds, evidence generation and upload |
| Checkout-free consumer smoke | 111789306289 | success | ~8 s | artifact-acquired basic template init/validate/doctor/build twice/inspect |
| Publish unsigned preview | 111789383802 | skipped | - | no publication dispatch; public alpha.65 unchanged |

The producer ran `Test release contracts` (`internal/releasebundle`,
`internal/releaseevidence` and `tests/hygiene` all `ok`), then
`Build release twice and generate evidence`, `Verify evidence documents` and
`Upload alpha evidence bundle`. The consumer checked out no source and
invoked only `velox.exe`; `consumer-clean.json` records
`acquisitionMode: github-actions-artifact-no-checkout` and
`invokedTools: ["velox.exe"]`.

## Verified Assets

| Asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,903,850 | `3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc` |
| `checksums.sha256` | 279 | `f20db6a79ad78170d83360b4b067d1024ca20220eaaae40c76c15c381728dd72` |
| `velox-windows-x64.spdx.json` | 9,618 | `196d958fbd52c8c68a10d75a7da7d9c6896fbb15ac5dbdb8b2fd46a2f3c26938` |
| `velox-windows-x64.intoto.jsonl` | 744 | `951fabec910eadb00d92a94fed0fa358ea328a39868fe846aee558410d3fe1c2` |

Release executables from `verified-result.json`, matching the manifest
entries:

| File | Bytes | SHA-256 |
| --- | --- | --- |
| `velox.exe` | 4,550,656 | `e04309c072c3d4ce96c2dae485e338b3dff225b91b352d6474edb4ccbbd8888b` |
| `velox-host.exe` | 4,416,000 | `7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb` |
| `velox-setup.exe` | 3,574,784 | `e0b40362dd0c82a8b31ea4922025683f952a372167d3086117fab1b44de86786` |

## Verification Detail

- Release manifest: 16 artifacts; every listed size and SHA-256 matched the ZIP
  bytes.
- SPDX: `SPDX-2.3`, one package, 17 file members including
  `release-manifest.json`; every SPDX digest was checked against the ZIP.
- Provenance: unsigned in-toto/SLSA v1 statement; subject
  `velox-windows-x64.zip` with the release digest, resolved dependency source
  `842fead0c889e9f161c2567a91c8d0fd4c2ca260`, builder
  `https://github.com/0disoft/velox/actions`, invocation `37317775399/1`,
  build type `.../alpha-evidence.yml@v1`.
- Checksums: the three `checksums.sha256` entries were recomputed locally and
  match the ZIP, SPDX and provenance files.
- Consumer: both builds produced app ZIP SHA-256
  `0ee5221c541312848255165c2dc8b0b1ef97bf157d4e17d4b33c7919f694fa33` and
  reported `deterministic: true`. The `endToEndMs` value 2559.566 ms is a
  basic hosted-fixture timing, not a performance comparison or a headline win.

## Controller Correction

The controller's first ad-hoc SPDX member count assumed 16 and omitted
`release-manifest.json`; it was corrected to verify all 17 members. No
artifact or source changed and the CI run was not rerun. The correction is
retained in `verified-result.json`.

## Not Covered By This Run

This run produced and consumed release bytes. It did not exercise native UI,
window startup, installer execution, `run --watch` or draft recovery against
these CI bytes. The earlier local source and native results are not upgraded
to this CI output.

## Public Status

- GitHub `/releases/tags/v0.5.10-alpha.66` returns HTTP 404, recorded in
  `publication-state.json` (`releaseLookup: "HTTP 404"`, `published: false`).
- The current public preview remains `v0.5.10-alpha.65`; its published hashes
  and run IDs are unchanged.
- Four byte-identical copies of the verified assets are staged flat under
  `dist/candidates/alpha66-ci-37317775399/publication/` for a later
  publication step. They can be published later without a rerun or rebuild
  once authorized. No publish or rebuild was performed for this record.
