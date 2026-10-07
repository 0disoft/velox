# Alpha.68 Tag CI Evidence: 2026-10-07

> Follow-up: these exact files were subsequently published without another
> producer; see [alpha68-publication.md](alpha68-publication.md). Public status
> below records the earlier tag-CI step, not the current published state.

## Identity

- Source: `5ffedc8d6e17b16e6e13ca8c57f21c06d76dfe8c`.
- Annotated tag `v0.5.10-alpha.68` and `main` were pushed atomically;
  remote main and the peeled tag matched this source at the push.
- [CI run 37628664660](https://github.com/0disoft/velox/actions/runs/37628664660):
  attempt 1, push event, success, 13:27:45Z-13:29:34Z.
- Actual toolchain: Go 1.26.0 Windows amd64. All three binaries record this
  source revision and `vcs.modified=false`; all three are `NotSigned`.
- These are not the previous local Go 1.27.1 candidate bytes.

## Jobs

| Job | ID | Result | Duration |
| --- | --- | --- | --- |
| Build reproducible unsigned preview evidence | 112816924513 | success | 91 s |
| Checkout-free consumer smoke | 112817624451 | success | 10 s |
| Publish unsigned preview | 112817729977 | skipped | - |

The producer passed releasebundle, releaseevidence and hygiene tests, then
two independent release builds matched. The consumer passed basic-template
init/validate/doctor/build twice/inspect using only `velox.exe`, with no
source checkout. No workflow dispatch, rerun or second producer was requested.

## Verified Assets

| Asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,923,846 | `b8e916c41c0d9fa6bc8ce21c6254e8adab8121c37d183da7a9e2cea99ba05bf8` |
| `checksums.sha256` | 279 | `b240f1193217d4bb69a5908cb304931b8cdcb56d1386e517d65b4b82e333fa4c` |
| `velox-windows-x64.spdx.json` | 9,618 | `5072791fdd29421385a79beba827ee84b13dbd5ab5dc12e80f8b6fe6746bc623` |
| `velox-windows-x64.intoto.jsonl` | 744 | `0a185eed5d1c81a2ac0fbf640ebbeaf347ab73a7b7718f6dc83dad05f68b88c3` |

| Binary | Bytes | SHA-256 |
| --- | --- | --- |
| CLI | 4,596,736 | `8509d901f4778f251eae249aed66f6e3265c0f98a93a9d948545110268851967` |
| Host | 4,416,000 | `2654d1551f889b46241fe58ba5e51eefd0547ad617566187a0602cf6c9fd0b65` |
| Setup | 3,574,784 | `d9765f2e1a40e659f02bb3a8b6e6bad438a4bd7d0f8d5c0dc422ebb591ec52de` |

The four downloaded assets passed local verification: all 16 manifest
entries, all 17 SPDX SHA-256 file entries, three sidecar checksums and
provenance source/run/ZIP identity matched. The exact CLI reported alpha.68;
PE subsystems were host/Setup GUI 2 and CLI console 3. Unsigned sidecars are
not authenticated attestations.

The consumer receipt binds the same source and release ZIP. Both app builds
had SHA-256 `3fed5e9e4aad6c2c7efa2c5ed2733b0a53276e129e0f83c11a49cf0ae8443da5`.
Its 4,447.924 ms end-to-end result is fixture timing, not a performance
comparison or an independent adoption result.

## Public Status And Boundaries

GitHub's alpha.68 release lookup returned HTTP 404. The public preview
remains published alpha.67. Four byte-identical files are staged flat under
`dist/candidates/alpha68-ci-37628664660/publication/` for a later publication
step, without another build. No release was created in this step.

Receipts are `run.json`, `ci-verification.json`, `publication-state.json` and
`consumer/consumer-clean.json` under `dist/candidates/alpha68-ci-37628664660/`.
The local verification script is `.cache/alpha68-ci-verify.mjs`; generated
receipts and assets are ignored and not committed.

No native UI, installer execution, watch/find/draft native repeat, hosted
stress or startup/memory comparison ran against these CI bytes. Earlier
local and manual results remain historical. No host/IPC API, DB/schema,
permissions, dependencies, repository hygiene or CI workflow change was
included. Public publication and beta promotion remain separate steps.
